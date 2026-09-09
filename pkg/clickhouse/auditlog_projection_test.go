package clickhouse

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestAuditLogProjection verifies that the logdrain payload query uses the
// cursor-ordered projection even though payload columns live in the base table.
func TestAuditLogProjection(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	table := "default." + uid.New("audit_projection")
	ddl, err := os.ReadFile("schema/025_audit_logs_raw_v1.sql")
	require.NoError(t, err)
	createTable := strings.Replace(string(ddl), "default.audit_logs_raw_v1", table, 1)
	require.NoError(t, client.conn.Exec(ctx, createTable))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(ctx, "DROP TABLE "+table)) })

	// The base key cannot prune insertion-time ranges across buckets.
	const rowsPerBucket = 8192
	const rowCount = 16 * rowsPerBucket
	insertedAt := time.Now().Add(-time.Hour).UnixMilli()
	insertLogs := `
		INSERT INTO ` + table + ` (workspace_id, bucket, event_id, time, inserted_at)
		SELECT
			'projection_workspace',
			toString(intDiv(number, ?)),
			concat('event_', leftPad(toString(number), 6, '0')),
			?,
			? + number
		FROM numbers(?)`
	require.NoError(t, client.conn.Exec(ctx, insertLogs, rowsPerBucket, insertedAt, insertedAt, rowCount))

	fromTime := insertedAt + 130000
	fromID := "event_130000"
	toExclusive := insertedAt + rowCount
	// Permit projection analysis for this small fixture without forcing selection.
	query := `
		SELECT
			event_id, time, inserted_at, event, description,
			actor_type, actor_id, actor_name,
			toJSONString(actor_meta) AS actor_meta_json,
			remote_ip, user_agent,
			toJSONString(meta) AS meta_json,
			targets.type AS target_types,
			targets.id AS target_ids,
			targets.name AS target_names,
			arrayMap(x -> toJSONString(x), targets.meta) AS target_metas_json,
			correlation_id
		FROM ` + table + `
		WHERE workspace_id = 'projection_workspace'
			AND (inserted_at > ? OR (inserted_at = ? AND event_id > ?))
			AND inserted_at < ?
		ORDER BY inserted_at, event_id LIMIT 1000
		SETTINGS min_table_rows_to_use_projection_index = 0`
	var plan []struct {
		Explain string `ch:"explain"`
	}
	require.NoError(t, client.conn.Select(ctx, &plan, "EXPLAIN projections=1, indexes=1 "+query, fromTime, fromTime, fromID, toExclusive))

	var explanation strings.Builder
	for _, line := range plan {
		explanation.WriteString(strings.TrimSpace(line.Explain) + "\n")
	}
	require.Contains(t, explanation.String(), "Name: proj_logdrain\n"+
		"Description: Projection has been analyzed and will be applied during reading")

	// Filtering-only projections are not listed in query_log.projections.
	// Compare actual reads with filtering off and on, without query-condition caching.
	readRows := func(projectionFiltering bool) uint64 {
		t.Helper()
		queryID := uid.New("query")
		queryCtx := ch.Context(ctx, ch.WithQueryID(queryID), ch.WithSettings(ch.Settings{
			"optimize_use_projection_filtering": projectionFiltering,
			"use_query_condition_cache":         false,
		}))
		rows, err := client.conn.Query(queryCtx, query, fromTime, fromTime, fromID, toExclusive)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, rows.Close()) })
		var returnedRows int
		for rows.Next() {
			returnedRows++
		}
		require.NoError(t, rows.Err())
		require.Equal(t, 1000, returnedRows)

		require.NoError(t, client.conn.Exec(ctx, "SYSTEM FLUSH LOGS"))
		var rowsRead uint64
		require.NoError(t, client.conn.QueryRow(ctx, `
			SELECT read_rows
			FROM system.query_log
			WHERE query_id = ? AND type = 'QueryFinish'
		`, queryID).Scan(&rowsRead))
		return rowsRead
	}

	rowsWithoutProjection := readRows(false)
	rowsWithProjection := readRows(true)
	require.Positive(t, rowsWithProjection)
	require.Less(t, rowsWithProjection, rowsWithoutProjection)
}
