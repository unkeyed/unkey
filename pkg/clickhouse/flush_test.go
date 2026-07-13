package clickhouse

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestInsertQuery(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		"INSERT INTO default.build_step_logs_v1 (`time`, `workspace_id`, `project_id`, `deployment_id`, `step_id`, `message`)",
		InsertQuery[schema.BuildStepLogV1]())

	// Source is tagged ch:"-" until the 20260612000000 migration has run
	// everywhere; see the comment on schema.KeyVerification.Source.
	query := InsertQuery[schema.KeyVerification]()
	require.NotContains(t, query, "source")
	require.Contains(t, query, "`request_id`")
}

// staleRow simulates a binary built before the table gained a column.
type staleRow struct {
	ID string `ch:"id"`
}

func (staleRow) Table() string         { return "default.flush_skew_test" }
func (staleRow) InsertColumns() string { return "`id`" }

// TestFlushToleratesUnknownTableColumns is the regression test for the
// canary incident where key_verifications_raw_v2 gained a `source` column
// before the frontline binary knew about it. With a bare "INSERT INTO
// <table>", AppendStruct failed with `missing destination name "source"`
// and every batch was dropped. With the explicit column list, the server
// fills the omitted column from its DEFAULT.
func TestFlushToleratesUnknownTableColumns(t *testing.T) {
	t.Parallel()
	chCfg := containers.ClickHouse(t)

	client, err := New(Config{URL: chCfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	ctx := context.Background()
	table := staleRow{}.Table()
	err = client.conn.Exec(ctx, fmt.Sprintf(`
		CREATE OR REPLACE TABLE %s (
			id String,
			source LowCardinality(String) DEFAULT 'api'
		) ENGINE = MergeTree ORDER BY id
	`, table))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE IF EXISTS "+table))
	})

	err = flush(client, ctx, []staleRow{{ID: "row_1"}, {ID: "row_2"}})
	require.NoError(t, err)

	rows, err := client.conn.Query(ctx, fmt.Sprintf("SELECT id, source FROM %s ORDER BY id", table))
	require.NoError(t, err)
	defer rows.Close()

	var got []staleRow
	for rows.Next() {
		var id, source string
		require.NoError(t, rows.Scan(&id, &source))
		require.Equal(t, "api", source, "omitted column must fall back to its DEFAULT")
		got = append(got, staleRow{ID: id})
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []staleRow{{ID: "row_1"}, {ID: "row_2"}}, got)
}
