package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCountKeysByWorkspaceReadRetry(t *testing.T) {
	for _, pool := range []struct {
		name string
		wrap func(*sql.DB) DBTX
	}{
		{name: "sql pool", wrap: func(db *sql.DB) DBTX { return db }},
		{name: "db replica", wrap: func(db *sql.DB) DBTX {
			return &Replica{db: db, mode: "ro", tags: sqlcomment.Disabled()}
		}},
		{name: "mysql replica", wrap: func(db *sql.DB) DBTX {
			return mysql.NewReplicaFromDB(db, "ro", sqlcomment.Disabled())
		}},
		{name: "mysql primary", wrap: func(db *sql.DB) DBTX {
			return mysql.NewReplicaFromDB(db, "rw", sqlcomment.Disabled())
		}},
	} {
		t.Run(pool.name, func(t *testing.T) {
			reset := readRetryReset(t)
			for _, failure := range []struct {
				name string
				step readRetryStep
			}{
				{name: "query error", step: readRetryStep{err: reset}},
				{name: "scan error from rows next", step: readRetryStep{nextErr: reset}},
			} {
				t.Run(failure.name, func(t *testing.T) {
					db, script := openReadRetryDB(t, []string{"count"},
						failure.step,
						readRetryStep{values: [][]driver.Value{{int64(42)}}},
					)

					count, err := Query.CountKeysByWorkspace(t.Context(), pool.wrap(db), uid.New(uid.WorkspacePrefix))
					require.NoError(t, err)
					require.Equal(t, int64(42), count)
					require.Equal(t, 2, script.calls)
				})
			}
		})
	}
}

func TestListDeploymentRegionsByIdsReadRetryDiscardsPartialRows(t *testing.T) {
	deploymentID := uid.New(uid.DeploymentPrefix)
	db, script := openReadRetryDB(t, []string{"deployment_id", "region"},
		readRetryStep{
			values:  [][]driver.Value{{deploymentID, "stale-region"}},
			nextErr: readRetryReset(t),
		},
		readRetryStep{values: [][]driver.Value{
			{deploymentID, "eu-west-1"},
			{deploymentID, "us-east-1"},
		}},
	)

	regions, err := Query.ListDeploymentRegionsByIds(t.Context(), db, ListDeploymentRegionsByIdsParams{
		WorkspaceID:   uid.New(uid.WorkspacePrefix),
		DeploymentIds: []string{deploymentID},
	})
	require.NoError(t, err)
	require.Equal(t, []ListDeploymentRegionsByIdsRow{
		{DeploymentID: deploymentID, Region: "eu-west-1"},
		{DeploymentID: deploymentID, Region: "us-east-1"},
	}, regions)
	require.Equal(t, 2, script.calls)
}

func TestReadRetrySkipsWritesAndLockingReads(t *testing.T) {
	for _, operation := range []struct {
		name string
		run  func(context.Context, DBTX) error
	}{
		{name: "delete key", run: func(ctx context.Context, db DBTX) error {
			return Query.DeleteKeyByID(ctx, db, uid.New(uid.KeyPrefix))
		}},
		{name: "find flag for update", run: func(ctx context.Context, db DBTX) error {
			_, err := Query.FindFlagBySlug(ctx, db, FindFlagBySlugParams{
				WorkspaceID: uid.New(uid.WorkspacePrefix),
				Slug:        uid.New(uid.TestPrefix),
			})
			return err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			reset := readRetryReset(t)
			db, script := openReadRetryDB(t, nil, readRetryStep{err: reset})

			err := operation.run(t.Context(), db)
			require.ErrorIs(t, err, reset)
			require.Equal(t, 1, script.calls)
		})
	}
}

func TestReadRetrySkipsTransactionsConnectionsAndUnknownWrappers(t *testing.T) {
	for _, kind := range []string{"transaction", "db traced transaction", "mysql traced transaction", "connection", "unknown wrapper"} {
		t.Run(kind, func(t *testing.T) {
			reset := readRetryReset(t)
			db, script := openReadRetryDB(t, []string{"count"}, readRetryStep{err: reset})
			var executor DBTX
			switch kind {
			case "transaction", "db traced transaction", "mysql traced transaction":
				var tx mysql.DBTx
				var err error
				switch kind {
				case "transaction":
					tx, err = db.BeginTx(t.Context(), nil)
				case "db traced transaction":
					replica := &Replica{db: db, mode: "rw", tags: sqlcomment.Disabled()}
					tx, err = replica.Begin(t.Context())
				case "mysql traced transaction":
					tx, err = mysql.NewReplicaFromDB(db, "rw", sqlcomment.Disabled()).Begin(t.Context())
				}
				require.NoError(t, err)
				t.Cleanup(func() {
					if err := tx.Rollback(); !errors.Is(err, sql.ErrTxDone) {
						require.NoError(t, err)
					}
				})
				executor = tx
			case "connection":
				conn, err := db.Conn(t.Context())
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, conn.Close()) })
				executor = conn
			case "unknown wrapper":
				executor = struct{ DBTX }{DBTX: db}
			}

			count, err := Query.CountKeysByWorkspace(t.Context(), executor, uid.New(uid.WorkspacePrefix))
			require.ErrorIs(t, err, reset)
			require.Zero(t, count)
			require.Equal(t, 1, script.calls)
		})
	}
}

func TestCountKeysByWorkspaceReadRetryStopsAfterThreeAttempts(t *testing.T) {
	reset := readRetryReset(t)
	db, script := openReadRetryDB(t, []string{"count"},
		readRetryStep{err: reset},
		readRetryStep{err: reset},
		readRetryStep{err: reset},
	)

	count, err := Query.CountKeysByWorkspace(t.Context(), db, uid.New(uid.WorkspacePrefix))
	require.ErrorIs(t, err, reset)
	require.Zero(t, count)
	require.Equal(t, 3, script.calls)
}

func TestReadRetryRespectsCancellation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		calls int
	}{
		{name: "before execution", calls: 0},
		{name: "before backoff", calls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if tt.calls == 0 {
				cancel()
			}
			db, _ := openReadRetryDB(t, nil)
			calls := 0
			_, err := mysql.ReadWithRetry(ctx, db, func() (int64, error) {
				calls++
				cancel()
				return 0, readRetryReset(t)
			})
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, tt.calls, calls)
		})
	}
}

func readRetryReset(t *testing.T) *mysqldriver.MySQLError {
	t.Helper()
	return &mysqldriver.MySQLError{
		Number:   1815,
		SQLState: [5]byte{'0', '8', 'S', '0', '1'},
		Message:  "connection reset by peer",
	}
}

func openReadRetryDB(t *testing.T, columns []string, steps ...readRetryStep) (*sql.DB, *readRetryScript) {
	t.Helper()
	script := &readRetryScript{columns: columns, steps: steps}
	db := sql.OpenDB(script)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, script
}

type readRetryStep struct {
	err     error
	values  [][]driver.Value
	nextErr error
}

type readRetryScript struct {
	columns []string
	steps   []readRetryStep
	calls   int
}

func (s *readRetryScript) Connect(context.Context) (driver.Conn, error) {
	return &readRetryConn{script: s}, nil
}

func (s *readRetryScript) Driver() driver.Driver {
	return s
}

func (s *readRetryScript) Open(string) (driver.Conn, error) {
	return nil, errors.New("use the connector")
}

func (s *readRetryScript) next() readRetryStep {
	s.calls++
	if s.calls > len(s.steps) {
		return readRetryStep{err: errors.New("unexpected database attempt")}
	}
	return s.steps[s.calls-1]
}

type readRetryConn struct {
	script *readRetryScript
}

func (c *readRetryConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}

func (c *readRetryConn) Close() error {
	return nil
}

func (c *readRetryConn) Begin() (driver.Tx, error) {
	return readRetryTx{}, nil
}

func (c *readRetryConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	step := c.script.next()
	if step.err != nil {
		return nil, step.err
	}
	return &readRetryRows{columns: c.script.columns, values: step.values, nextErr: step.nextErr}, nil
}

func (c *readRetryConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	step := c.script.next()
	if step.err != nil {
		return nil, step.err
	}
	return driver.RowsAffected(1), nil
}

type readRetryTx struct{}

func (readRetryTx) Commit() error {
	return nil
}

func (readRetryTx) Rollback() error {
	return nil
}

type readRetryRows struct {
	columns []string
	values  [][]driver.Value
	nextErr error
}

func (r *readRetryRows) Columns() []string {
	return r.columns
}

func (r *readRetryRows) Close() error {
	return nil
}

func (r *readRetryRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		if r.nextErr != nil {
			return r.nextErr
		}
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}
