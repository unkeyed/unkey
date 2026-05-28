package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/sqlcommenter"
)

// TestSQLCommenter_AppearsOnWire verifies that comments built by pkg/sqlcommenter
// reach the underlying driver when a query runs through a Replica. Without this,
// the formatter could be perfect but the Replica wiring could be silently
// dropping it — only an end-to-end check catches that.
//
// Uses a fake driver instead of MySQL so the test stays fast and deterministic.
func TestSQLCommenter_AppearsOnWire(t *testing.T) {
	rec := &recordingDriver{}
	sqlDB := openWithDriver(t, rec)

	r := &Replica{db: sqlDB, mode: "rw", application: "test-svc"}

	ctx := sqlcommenter.WithRequest(context.Background(), "POST /v1/keys.createKey", "req_abc123")

	rows, err := r.QueryContext(ctx, "SELECT 1")
	require.NoError(t, err)
	require.NoError(t, rows.Close())

	captured := rec.lastQuery()
	require.Contains(t, captured, "application='test-svc'")
	require.Contains(t, captured, "request_id='req_abc123'")
	require.Contains(t, captured, "route='POST%20%2Fv1%2Fkeys.createKey'")
	require.Contains(t, captured, "db_driver='go-database-sql'")
}

// TestSQLCommenter_NotAppendedToPrepare guards the prepared-statement cache:
// per-request comments on PREPARE'd queries would defeat both the client- and
// server-side caches.
func TestSQLCommenter_NotAppendedToPrepare(t *testing.T) {
	rec := &recordingDriver{}
	sqlDB := openWithDriver(t, rec)

	r := &Replica{db: sqlDB, mode: "rw", application: "test-svc"}

	ctx := sqlcommenter.WithRequest(context.Background(), "/test", "req_prepare")

	stmt, err := r.PrepareContext(ctx, "SELECT 2")
	require.NoError(t, err)
	t.Cleanup(func() { _ = stmt.Close() })

	captured := rec.lastPrepare()
	require.NotContains(t, captured, "application='", "PREPARE must not carry per-request sqlcommenter tags")
	require.NotContains(t, captured, "request_id='")
}

// recordingDriver implements just enough of database/sql/driver to capture the
// raw SQL strings the Replica passes through. Registered once per test under
// a unique name so parallel tests don't collide.
type recordingDriver struct {
	mu       sync.Mutex
	queries  []string
	prepares []string
}

func (d *recordingDriver) Open(_ string) (driver.Conn, error) { return &recordingConn{drv: d}, nil }

func (d *recordingDriver) lastQuery() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.queries) == 0 {
		return ""
	}
	return d.queries[len(d.queries)-1]
}

func (d *recordingDriver) lastPrepare() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.prepares) == 0 {
		return ""
	}
	return d.prepares[len(d.prepares)-1]
}

type recordingConn struct{ drv *recordingDriver }

func (c *recordingConn) Prepare(query string) (driver.Stmt, error) {
	c.drv.mu.Lock()
	c.drv.prepares = append(c.drv.prepares, query)
	c.drv.mu.Unlock()
	return &recordingStmt{}, nil
}
func (c *recordingConn) Close() error              { return nil }
func (c *recordingConn) Begin() (driver.Tx, error) { return &recordingTx{}, nil }

func (c *recordingConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.drv.mu.Lock()
	c.drv.queries = append(c.drv.queries, query)
	c.drv.mu.Unlock()
	return &emptyRows{}, nil
}

func (c *recordingConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.drv.mu.Lock()
	c.drv.queries = append(c.drv.queries, query)
	c.drv.mu.Unlock()
	return driver.RowsAffected(0), nil
}

type recordingStmt struct{}

func (s *recordingStmt) Close() error  { return nil }
func (s *recordingStmt) NumInput() int { return -1 }
func (s *recordingStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (s *recordingStmt) Query(_ []driver.Value) (driver.Rows, error) { return &emptyRows{}, nil }

type recordingTx struct{}

func (t *recordingTx) Commit() error   { return nil }
func (t *recordingTx) Rollback() error { return nil }

type emptyRows struct{}

func (r *emptyRows) Columns() []string           { return nil }
func (r *emptyRows) Close() error                { return nil }
func (r *emptyRows) Next(_ []driver.Value) error { return io.EOF }

var registerMu sync.Mutex
var registerCounter int

func openWithDriver(t *testing.T, d driver.Driver) *sql.DB {
	t.Helper()

	registerMu.Lock()
	registerCounter++
	name := newDriverName(registerCounter)
	registerMu.Unlock()

	sql.Register(name, d)
	sqlDB, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

func newDriverName(n int) string {
	return "sqlcommenter_test_" + strconv.Itoa(n)
}
