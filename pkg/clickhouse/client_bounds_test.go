package clickhouse

import (
	"context"
	"errors"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
)

type queryResultConnectionFake struct {
	driver.Conn
	rows     driver.Rows
	ctxQuery context.Context
}

func (c *queryResultConnectionFake) Query(ctx context.Context, _ string, _ ...any) (driver.Rows, error) {
	c.ctxQuery = ctx
	if rows, ok := c.rows.(*queryResultRowsFake); ok {
		rows.ctxQuery = ctx
	}
	return c.rows, nil
}

type queryResultRowsFake struct {
	columns         []string
	rows            [][]any
	index           int
	closed          bool
	errContextClose error
	ctxQuery        context.Context
	err             error
}

func (r *queryResultRowsFake) Next() bool {
	return r.index < len(r.rows)
}

func (r *queryResultRowsFake) Scan(destinations ...any) error {
	for i, value := range r.rows[r.index] {
		dynamic, ok := destinations[i].(*ch.Dynamic)
		if !ok {
			return errors.New("unexpected scan destination")
		}
		*dynamic = ch.NewDynamic(value)
	}
	r.index++
	return nil
}

func (r *queryResultRowsFake) ScanStruct(any) error             { return errors.New("not implemented") }
func (r *queryResultRowsFake) ColumnTypes() []driver.ColumnType { return nil }
func (r *queryResultRowsFake) Totals(...any) error              { return nil }
func (r *queryResultRowsFake) Columns() []string                { return r.columns }
func (r *queryResultRowsFake) Close() error {
	r.closed = true
	r.errContextClose = r.ctxQuery.Err()
	return nil
}
func (r *queryResultRowsFake) Err() error { return r.err }

// Security guarantee: a single aggregate row is subject to the byte budget, not only the row cap.
func TestQueryToMapsRejectsOneHugeAggregate(t *testing.T) {
	rows := &queryResultRowsFake{columns: []string{"groupArray(payload)"}, rows: [][]any{{string(make([]byte, 1024))}}}
	client := &Client{conn: &queryResultConnectionFake{rows: rows}}

	_, err := client.QueryToMaps(context.Background(), "SELECT groupArray(payload)", QueryResultLimits{RowsMax: 10, BytesMax: 128})
	require.ErrorContains(t, err, "result byte limit")
	require.True(t, rows.closed)
}

// Security guarantee: wide rows consume the same byte budget as many narrow rows.
func TestQueryToMapsRejectsWideRows(t *testing.T) {
	rows := &queryResultRowsFake{columns: []string{"a", "b", "c"}, rows: [][]any{{string(make([]byte, 60)), string(make([]byte, 60)), string(make([]byte, 60))}}}
	client := &Client{conn: &queryResultConnectionFake{rows: rows}}

	_, err := client.QueryToMaps(context.Background(), "SELECT a, b, c", QueryResultLimits{RowsMax: 10, BytesMax: 128})
	require.ErrorContains(t, err, "result byte limit")
	require.True(t, rows.closed)
}

// Security guarantee: stopping response consumption closes ClickHouse rows and cancels further delivery.
func TestQueryToMapsClosesRowsAtHardRowCap(t *testing.T) {
	rows := &queryResultRowsFake{columns: []string{"n"}, rows: [][]any{{1}, {2}}}
	connection := &queryResultConnectionFake{rows: rows}
	client := &Client{conn: connection}

	_, err := client.QueryToMaps(context.Background(), "SELECT n", QueryResultLimits{RowsMax: 1, BytesMax: 1024})
	require.ErrorContains(t, err, "result row limit")
	require.True(t, rows.closed)
	require.ErrorIs(t, connection.ctxQuery.Err(), context.Canceled)
	require.ErrorIs(t, rows.errContextClose, context.Canceled)
}

// Security guarantee: callers cannot accidentally execute an unbounded dynamic query.
func TestQueryToMapsRejectsDisabledBounds(t *testing.T) {
	_, err := NewNoop().QueryToMaps(context.Background(), "SELECT 1", QueryResultLimits{RowsMax: 0, BytesMax: 0})
	require.ErrorContains(t, err, "query result row limit must be positive")
}

// Security guarantee: the API enforces the positive workspace quota without silently replacing it with a separate row policy.
func TestQueryToMapsUsesWorkspaceRowBound(t *testing.T) {
	_, err := NewNoop().QueryToMaps(context.Background(), "SELECT 1", QueryResultLimits{
		RowsMax:  10_000_000,
		BytesMax: AnalyticsResultBytesMax,
	})
	require.NoError(t, err)
}

// Security guarantee: callers cannot weaken the global encoded-response bound.
func TestQueryToMapsRejectsByteBoundAboveHardMaximum(t *testing.T) {
	_, err := NewNoop().QueryToMaps(context.Background(), "SELECT 1", QueryResultLimits{
		RowsMax:  10_000_000,
		BytesMax: AnalyticsResultBytesMax + 1,
	})
	require.ErrorContains(t, err, "query result byte limit exceeds the hard maximum")
}
