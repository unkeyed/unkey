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

func TestQueryToMapsRejectsOneHugeAggregate(t *testing.T) {
	// Security guarantee: a single aggregate row is subject to the byte budget, not only the row cap.
	rows := &boundedTestRows{columns: []string{"groupArray(payload)"}, rows: [][]any{{string(make([]byte, 1024))}}}
	client := &Client{conn: &boundedTestConn{rows: rows}}

	_, err := client.QueryToMaps(context.Background(), "SELECT groupArray(payload)", QueryResultLimits{RowsMax: 10, BytesMax: 128})
	require.ErrorContains(t, err, "result byte limit")
	require.True(t, rows.closed)
}

func TestQueryToMapsRejectsWideRows(t *testing.T) {
	// Security guarantee: wide rows consume the same byte budget as many narrow rows.
	rows := &boundedTestRows{columns: []string{"a", "b", "c"}, rows: [][]any{{string(make([]byte, 60)), string(make([]byte, 60)), string(make([]byte, 60))}}}
	client := &Client{conn: &boundedTestConn{rows: rows}}

	_, err := client.QueryToMaps(context.Background(), "SELECT a, b, c", QueryResultLimits{RowsMax: 10, BytesMax: 128})
	require.ErrorContains(t, err, "result byte limit")
	require.True(t, rows.closed)
}

func TestQueryToMapsClosesRowsAtHardRowCap(t *testing.T) {
	// Security guarantee: stopping response consumption closes ClickHouse rows and cancels further delivery.
	rows := &boundedTestRows{columns: []string{"n"}, rows: [][]any{{1}, {2}}}
	conn := &boundedTestConn{rows: rows}
	client := &Client{conn: conn}

	_, err := client.QueryToMaps(context.Background(), "SELECT n", QueryResultLimits{RowsMax: 1, BytesMax: 1024})
	require.ErrorContains(t, err, "result row limit")
	require.True(t, rows.closed)
	require.ErrorIs(t, conn.queryCtx.Err(), context.Canceled)
	require.ErrorIs(t, rows.closeContextErr, context.Canceled)
}

func TestQueryToMapsRejectsDisabledBounds(t *testing.T) {
	// Security guarantee: callers cannot accidentally execute an unbounded dynamic query.
	_, err := NewNoop().QueryToMaps(context.Background(), "SELECT 1", QueryResultLimits{RowsMax: 0, BytesMax: 0})
	require.ErrorContains(t, err, "query result row limit must be positive")
}

func TestQueryToMapsRejectsBoundsAboveHardMaximums(t *testing.T) {
	// Security guarantee: callers cannot weaken global analytics bounds with permissive per-query values.
	tests := []struct {
		name   string
		limits QueryResultLimits
		err    string
	}{
		{
			name:   "rows",
			limits: QueryResultLimits{RowsMax: AnalyticsResultRowsMax + 1, BytesMax: AnalyticsResultBytesMax},
			err:    "query result row limit exceeds the hard maximum",
		},
		{
			name:   "bytes",
			limits: QueryResultLimits{RowsMax: AnalyticsResultRowsMax, BytesMax: AnalyticsResultBytesMax + 1},
			err:    "query result byte limit exceeds the hard maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNoop().QueryToMaps(context.Background(), "SELECT 1", tt.limits)
			require.ErrorContains(t, err, tt.err)
		})
	}
}
