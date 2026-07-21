package clickhouse

import (
	"context"
	"errors"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
)

type boundedTestConn struct {
	driver.Conn
	rows     driver.Rows
	queryCtx context.Context
}

func (c *boundedTestConn) Query(ctx context.Context, _ string, _ ...any) (driver.Rows, error) {
	c.queryCtx = ctx
	if rows, ok := c.rows.(*boundedTestRows); ok {
		rows.queryCtx = ctx
	}
	return c.rows, nil
}

type boundedTestRows struct {
	columns         []string
	rows            [][]any
	index           int
	closed          bool
	closeContextErr error
	queryCtx        context.Context
	err             error
}

func (r *boundedTestRows) Next() bool {
	return r.index < len(r.rows)
}

func (r *boundedTestRows) Scan(dest ...any) error {
	for i, value := range r.rows[r.index] {
		ptr, ok := dest[i].(*ch.Dynamic)
		if !ok {
			return errors.New("unexpected scan destination")
		}
		*ptr = ch.NewDynamic(value)
	}
	r.index++
	return nil
}

func (r *boundedTestRows) ScanStruct(any) error             { return errors.New("not implemented") }
func (r *boundedTestRows) ColumnTypes() []driver.ColumnType { return nil }
func (r *boundedTestRows) Totals(...any) error              { return nil }
func (r *boundedTestRows) Columns() []string                { return r.columns }
func (r *boundedTestRows) Close() error {
	r.closed = true
	r.closeContextErr = r.queryCtx.Err()
	return nil
}
func (r *boundedTestRows) Err() error { return r.err }

func TestQueryToMapsBoundedRejectsOneHugeAggregate(t *testing.T) {
	// Security guarantee: a single aggregate row is subject to the byte budget, not only the row cap.
	rows := &boundedTestRows{columns: []string{"groupArray(payload)"}, rows: [][]any{{string(make([]byte, 1024))}}}
	client := &Client{conn: &boundedTestConn{rows: rows}}

	_, err := client.QueryToMapsBounded(context.Background(), "SELECT groupArray(payload)", ResultLimits{MaxRows: 10, MaxBytes: 128})
	require.ErrorContains(t, err, "result byte limit")
	require.True(t, rows.closed)
}

func TestQueryToMapsBoundedRejectsWideRows(t *testing.T) {
	// Security guarantee: wide rows consume the same byte budget as many narrow rows.
	rows := &boundedTestRows{columns: []string{"a", "b", "c"}, rows: [][]any{{string(make([]byte, 60)), string(make([]byte, 60)), string(make([]byte, 60))}}}
	client := &Client{conn: &boundedTestConn{rows: rows}}

	_, err := client.QueryToMapsBounded(context.Background(), "SELECT a, b, c", ResultLimits{MaxRows: 10, MaxBytes: 128})
	require.ErrorContains(t, err, "result byte limit")
	require.True(t, rows.closed)
}

func TestQueryToMapsBoundedClosesRowsAtHardRowCap(t *testing.T) {
	// Security guarantee: stopping response consumption closes ClickHouse rows and cancels further delivery.
	rows := &boundedTestRows{columns: []string{"n"}, rows: [][]any{{1}, {2}}}
	conn := &boundedTestConn{rows: rows}
	client := &Client{conn: conn}

	_, err := client.QueryToMapsBounded(context.Background(), "SELECT n", ResultLimits{MaxRows: 1, MaxBytes: 1024})
	require.ErrorContains(t, err, "result row limit")
	require.True(t, rows.closed)
	require.ErrorIs(t, conn.queryCtx.Err(), context.Canceled)
	require.ErrorIs(t, rows.closeContextErr, context.Canceled)
}
