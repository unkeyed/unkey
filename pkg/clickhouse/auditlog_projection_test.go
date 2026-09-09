package clickhouse

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

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

	fromTime := insertedAt + 1000
	fromID := "event_001000"
	toExclusive := insertedAt + 2072
	// Permit projection analysis for this small fixture without forcing selection.
	query := `
		EXPLAIN projections=1, indexes=1
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
			AND (inserted_at, event_id) > (?, ?)
			AND inserted_at < ?
		ORDER BY inserted_at, event_id LIMIT 1000
		SETTINGS min_table_rows_to_use_projection_index = 0`
	var plan []struct {
		Explain string `ch:"explain"`
	}
	require.NoError(t, client.conn.Select(ctx, &plan, query, fromTime, fromID, toExclusive))

	var explanation strings.Builder
	for _, line := range plan {
		explanation.WriteString(strings.TrimSpace(line.Explain) + "\n")
	}
	require.Contains(t, explanation.String(), "Name: proj_logdrain\n"+
		"Description: Projection has been analyzed and will be applied during reading")
}
