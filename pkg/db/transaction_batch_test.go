package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
)

type ambiguousBatchDriver struct {
	conn *ambiguousBatchConn
}

func (d ambiguousBatchDriver) Open(string) (driver.Conn, error) {
	return d.conn, nil
}

type ambiguousBatchConn struct {
	queryCalls    int
	rollbackCalls int
	closed        bool
	rows          driver.Rows
	queryErr      error
}

func (c *ambiguousBatchConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *ambiguousBatchConn) Close() error {
	c.closed = true
	return nil
}

func (c *ambiguousBatchConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c *ambiguousBatchConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	c.queryCalls++
	return c.rows, c.queryErr
}

type markerThenErrorRows struct {
	next int
}

func (*markerThenErrorRows) Columns() []string { return []string{"marker"} }
func (*markerThenErrorRows) Close() error      { return nil }
func (r *markerThenErrorRows) Next(dest []driver.Value) error {
	if r.next == 0 {
		r.next++
		dest[0] = transactionBatchMarker
		return nil
	}
	if r.next == 1 {
		r.next++
		return errors.New("connection lost while draining")
	}
	return io.EOF
}

func (c *ambiguousBatchConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if query == "ROLLBACK" {
		c.rollbackCalls++
	}
	return driver.RowsAffected(0), nil
}

func TestUpdateKeyBatchAnnotatesEveryStatement(t *testing.T) {
	t.Parallel()

	replica := &Replica{
		mode:      "rw",
		db:        nil,
		debugLogs: false,
		tags:      sqlcomment.ForService("api", "us-east-1"),
	}
	query, _ := replica.transactionBatchQuery(t.Context(), "UpdateKeyWithAuditBatch", []transactionBatchStatement{
		{query: updateKey},
		{query: insertClickhouseOutbox},
	})
	statements := strings.Split(strings.TrimSuffix(query, ";"), ";\n")
	require.Len(t, statements, 5)

	operations := []string{
		"UpdateKeyWithAuditBatchStart",
		"UpdateKey",
		"InsertClickhouseOutbox",
		"UpdateKeyWithAuditBatchCommit",
		"UpdateKeyWithAuditBatchMarker",
	}
	for i, statement := range statements {
		require.NotContains(t, statement, "-- name:")
		require.Contains(t, statement, "service='api'")
		require.Contains(t, statement, "mode='rw'")
		require.Contains(t, statement, "operation='"+operations[i]+"'")
	}
}

func TestUpdateKeyBatchDoesNotReplayAmbiguousCommit(t *testing.T) {
	conn := &ambiguousBatchConn{queryErr: errors.New("connection lost before commit marker")}
	const driverName = "unkey-ambiguous-update-key-batch-test"
	sql.Register(driverName, ambiguousBatchDriver{conn: conn})

	sqlDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	replica := &Replica{mode: "rw", db: sqlDB, tags: sqlcomment.Disabled()}

	err = replica.UpdateKeyWithAuditBatch(t.Context(), UpdateKeyParams{
		ID:            "key_test",
		NameSpecified: 1,
		Name:          sql.NullString{String: "after", Valid: true},
		Now:           sql.NullInt64{Int64: 1, Valid: true},
	}, InsertClickhouseOutboxParams{
		Version:     "audit_log.v1",
		WorkspaceID: "ws_test",
		EventID:     "evt_test",
		Payload:     []byte(`{"event":"key.update"}`),
		CreatedAt:   1,
	})
	require.ErrorContains(t, err, "connection lost before commit marker")
	require.Equal(t, 1, conn.queryCalls, "an ambiguous mutation must not be replayed")
	require.Equal(t, 1, conn.rollbackCalls)
	require.True(t, conn.closed, "the physical connection must be discarded")
}

func TestUpdateKeyBatchReturnsSuccessAfterMarker(t *testing.T) {
	conn := &ambiguousBatchConn{rows: &markerThenErrorRows{}}
	const driverName = "unkey-committed-update-key-batch-test"
	sql.Register(driverName, ambiguousBatchDriver{conn: conn})

	sqlDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	replica := &Replica{mode: "rw", db: sqlDB, tags: sqlcomment.Disabled()}

	err = replica.UpdateKeyWithAuditBatch(t.Context(), UpdateKeyParams{
		ID:  "key_test",
		Now: sql.NullInt64{Int64: 1, Valid: true},
	}, InsertClickhouseOutboxParams{
		Version:     "audit_log.v1",
		WorkspaceID: "ws_test",
		EventID:     "evt_test",
		Payload:     []byte(`{"event":"key.update"}`),
		CreatedAt:   1,
	})
	require.NoError(t, err, "the marker proves that the mutation committed")
	require.Equal(t, 1, conn.queryCalls)
	require.Equal(t, 0, conn.rollbackCalls)
	require.True(t, conn.closed, "a connection with a drain error must be discarded")
}
