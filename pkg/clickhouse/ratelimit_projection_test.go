package clickhouse

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestRatelimitProjection_FullPayload(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := t.Context()
	table := "default." + uid.New("ratelimit_projection")
	ddl, err := os.ReadFile("schema/006_ratelimits_raw_v2.sql")
	require.NoError(t, err)
	require.NoError(t, client.conn.Exec(ctx, strings.Replace(string(ddl), "ratelimits_raw_v2", table, 1)))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE "+table)) })
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+`
		(workspace_id, request_id, check_index, time, inserted_at, namespace_id, identifier, passed, override_id, limit, remaining, reset_at, tokens)
		SELECT 'workspace', toString(number), 0, ? - number, ? + number, toString(number % 3), 'customer', number % 2 = 0, 'override', 100, 7, 123456, 3 FROM numbers(1048576)`, now, now))
	query := `SELECT inserted_at, time, event_id, request_id, check_index, namespace_id, identifier, passed, override_id, limit, remaining, reset_at, tokens
		FROM ` + table + ` WHERE workspace_id = 'workspace'
		AND (inserted_at > {from_time:Int64} OR (inserted_at = {from_time:Int64} AND event_id > '1000000:0'))
		AND inserted_at < {to:Int64}
		AND (empty({namespaces:Array(String)}) OR namespace_id IN {namespaces:Array(String)})
		AND (empty({passed:Array(Bool)}) OR passed IN {passed:Array(Bool)})
		ORDER BY inserted_at, event_id LIMIT 1000`
	for _, tt := range []struct {
		name, namespaces, passed, first string
		count                           int
	}{
		{"all", "[]", "[]", "1000001", 1000},
		{"combined", "['1']", "[true]", "1000006", 166},
	} {
		t.Run(tt.name, func(t *testing.T) {
			queryCtx := ch.Context(ctx, ch.WithParameters(map[string]string{
				"from_time": strconv.FormatInt(now+1000000, 10), "to": strconv.FormatInt(now+1001001, 10),
				"namespaces": tt.namespaces, "passed": tt.passed,
			}))
			var plan []struct {
				Explain string `ch:"explain"`
			}
			require.NoError(t, client.conn.Select(queryCtx, &plan, "EXPLAIN projections=1, indexes=1 "+query))
			var explanation strings.Builder
			for _, line := range plan {
				explanation.WriteString(strings.TrimSpace(line.Explain) + "\n")
			}
			require.Contains(t, explanation.String(), "Name: proj_logdrain\nDescription: Projection has been analyzed and will be applied during reading")
			for _, enabled := range []bool{false, true} {
				id := uid.New("query")
				readCtx := ch.Context(queryCtx, ch.WithQueryID(id), ch.WithSettings(ch.Settings{"optimize_use_projection_filtering": enabled, "use_query_condition_cache": false}))
				rows, err := client.conn.Query(readCtx, query)
				require.NoError(t, err)
				count := 0
				for rows.Next() {
					var insertedAt, eventTime, resetAt int64
					var eventID, requestID, namespace, identifier, override string
					var index uint32
					var passed bool
					var limit, remaining, tokens uint64
					require.NoError(t, rows.Scan(&insertedAt, &eventTime, &eventID, &requestID, &index, &namespace, &identifier, &passed, &override, &limit, &remaining, &resetAt, &tokens))
					if count == 0 {
						require.Equal(t, tt.first, requestID)
					}
					require.Equal(t, requestID+":0", eventID)
					require.Zero(t, index)
					require.Equal(t, "customer", identifier)
					require.Equal(t, "override", override)
					require.EqualValues(t, 100, limit)
					require.EqualValues(t, 7, remaining)
					require.EqualValues(t, 123456, resetAt)
					require.EqualValues(t, 3, tokens)
					require.Equal(t, 2*now, insertedAt+eventTime)
					if tt.name == "combined" {
						require.True(t, passed)
						require.Equal(t, "1", namespace)
					}
					count++
				}
				require.NoError(t, rows.Err())
				require.NoError(t, rows.Close())
				require.Equal(t, tt.count, count)
				require.NoError(t, client.conn.Exec(ctx, "SYSTEM FLUSH LOGS"))
				var readRows, readBytes uint64
				require.NoError(t, client.conn.QueryRow(ctx, `SELECT read_rows, read_bytes FROM system.query_log WHERE query_id = ? AND type = 'QueryFinish'`, id).Scan(&readRows, &readBytes))
				t.Logf("default projection threshold; enabled=%t rows=%d bytes=%d", enabled, readRows, readBytes)
			}
		})
	}
}

func TestRatelimitMigration_FreezesHistory(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := t.Context()
	table := "default." + uid.New("ratelimit_migration")
	require.NoError(t, client.conn.Exec(ctx, `CREATE TABLE `+table+`
		(request_id String, time Int64, workspace_id String, namespace_id String, identifier String, passed Bool)
		ENGINE = MergeTree ORDER BY (workspace_id, time, namespace_id)
		TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 1 MONTH DELETE
		SETTINGS non_replicated_deduplication_window = 10000`))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE "+table)) })
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+` (workspace_id, request_id, time) VALUES ('workspace', 'old', ?), ('workspace', 'old', ?)`, now, now))
	migration, err := os.ReadFile("migrations/20260910030000.sql")
	require.NoError(t, err)
	for _, statement := range strings.Split(string(migration), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		require.NoError(t, client.conn.Exec(ctx, strings.ReplaceAll(statement, "default.ratelimits_raw_v2", table)))
	}
	var historical []struct {
		InsertedAt int64 `ch:"inserted_at"`
		Time       int64 `ch:"time"`
	}
	require.NoError(t, client.conn.Select(ctx, &historical, "SELECT inserted_at, time FROM "+table))
	require.Len(t, historical, 2)
	for _, row := range historical {
		require.Zero(t, row.InsertedAt)
		require.Equal(t, now, row.Time)
	}
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+` (workspace_id, request_id, check_index, time) VALUES ('workspace', 'new', 0, ?), ('workspace', 'new', 1, ?)`, now-60000, now-60000))
	var insertedAt int64
	var ids uint64
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT min(inserted_at), uniqExact(event_id) FROM `+table+` WHERE request_id = 'new'`).Scan(&insertedAt, &ids))
	require.GreaterOrEqual(t, insertedAt, now)
	require.EqualValues(t, 2, ids)
	var ddl string
	require.NoError(t, client.conn.QueryRow(ctx, "SHOW CREATE TABLE "+table).Scan(&ddl))
	require.Contains(t, ddl, "TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalMonth(1)")
	require.Contains(t, ddl, "non_replicated_deduplication_window = 10000")
}
