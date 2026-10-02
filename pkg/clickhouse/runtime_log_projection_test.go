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

func TestRuntimeLogProjection(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := t.Context()
	table := "default." + uid.New("runtime_projection")
	ddl, err := os.ReadFile("schema/023_runtime_logs_raw_v1.sql")
	require.NoError(t, err)
	require.NoError(t, client.conn.Exec(ctx, strings.Replace(string(ddl), "default.runtime_logs_raw_v1", table, 1)))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE "+table)) })
	const rowCount = 131072
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+`
		(workspace_id, log_id, time, inserted_at, project_id, severity,
		app_id, environment_id, deployment_id, region, message, attributes)
		SELECT 'workspace', leftPad(toString(number), 6, '0'),
		? - number, ? + number, toString(number % 32), if(number % 2 = 0, 'info', 'error'),
		'app', 'env', 'deployment', 'eu-west-1', concat('Order ', toString(number)),
		toJSONString(map('order', map('id', number))) FROM numbers(?)`, now, now, rowCount))
	query := `SELECT inserted_at, time, log_id, severity, message,
		toJSONString(attributes), project_id, app_id, environment_id, deployment_id, region
		FROM ` + table + ` WHERE workspace_id = 'workspace'
		AND (inserted_at > {from_time:Int64} OR (inserted_at = {from_time:Int64} AND log_id > '130000'))
		AND inserted_at < {to:Int64}
		AND (empty({severities:Array(String)}) OR severity IN {severities:Array(String)})
		AND (empty({projects:Array(String)}) OR project_id IN {projects:Array(String)})
		AND (empty({apps:Array(String)}) OR app_id IN {apps:Array(String)})
		AND (empty({environments:Array(String)}) OR environment_id IN {environments:Array(String)})
		ORDER BY inserted_at, log_id LIMIT 1000
		SETTINGS min_table_rows_to_use_projection_index = 0`
	for _, tt := range []struct {
		name, severities, projects, apps, environments string
		wantRows                                       int
	}{
		{"all", "[]", "[]", "[]", "[]", 1000},
		{"combined", "['info']", "['16']", "['app']", "['env']", 33},
	} {
		t.Run(tt.name, func(t *testing.T) {
			queryCtx := ch.Context(ctx, ch.WithParameters(map[string]string{
				"from_time": strconv.FormatInt(now+130000, 10), "to": strconv.FormatInt(now+rowCount, 10),
				"severities": tt.severities, "projects": tt.projects, "apps": tt.apps, "environments": tt.environments,
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
				readCtx := ch.Context(queryCtx, ch.WithQueryID(id), ch.WithSettings(ch.Settings{
					"optimize_use_projection_filtering": enabled, "use_query_condition_cache": false,
				}))
				rows, err := client.conn.Query(readCtx, query)
				require.NoError(t, err)
				count := 0
				for rows.Next() {
					count++
				}
				require.NoError(t, rows.Err())
				require.NoError(t, rows.Close())
				require.Equal(t, tt.wantRows, count)
				require.NoError(t, client.conn.Exec(ctx, "SYSTEM FLUSH LOGS"))
				var readRows, readBytes uint64
				require.NoError(t, client.conn.QueryRow(ctx, `SELECT read_rows, read_bytes FROM system.query_log WHERE query_id = ? AND type = 'QueryFinish'`, id).Scan(&readRows, &readBytes))
				t.Logf("diagnostic min_table_rows_to_use_projection_index=0; projection=%t read_rows=%d read_bytes=%d", enabled, readRows, readBytes)
			}
		})
	}
}

func TestRuntimeLogMigration_PreservesExistingRows(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := t.Context()
	table := "default." + uid.New("runtime_migration")
	ddl, err := os.ReadFile("schema/023_runtime_logs_raw_v1.sql")
	require.NoError(t, err)
	require.NoError(t, client.conn.Exec(ctx, strings.Replace(string(ddl), "default.runtime_logs_raw_v1", table, 1)))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE "+table)) })
	require.NoError(t, client.conn.Exec(ctx, "ALTER TABLE "+table+" DROP PROJECTION proj_logdrain"))
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, "INSERT INTO "+table+" (workspace_id, log_id, inserted_at, time, expires_at) VALUES ('workspace', 'old', ?, ?, fromUnixTimestamp64Milli(?))", now-3600000, now-7200000, now+86400000))
	migration, err := os.ReadFile("migrations/20260910020000.sql")
	require.NoError(t, err)
	for _, statement := range strings.Split(string(migration), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		require.NoError(t, client.conn.Exec(ctx, strings.ReplaceAll(statement, "default.runtime_logs_raw_v1", table)))
	}
	var insertedAt, eventTime, expiresAt int64
	require.NoError(t, client.conn.QueryRow(ctx, "SELECT inserted_at, time, toUnixTimestamp64Milli(expires_at) FROM "+table+" WHERE log_id = 'old'").Scan(&insertedAt, &eventTime, &expiresAt))
	require.Equal(t, now-3600000, insertedAt)
	require.Equal(t, now-7200000, eventTime)
	require.Equal(t, now+86400000, expiresAt)
	require.NoError(t, client.conn.Exec(ctx, "INSERT INTO "+table+" (workspace_id, log_id, time) VALUES ('workspace', 'new', ?)", now-60000))
	require.NoError(t, client.conn.QueryRow(ctx, "SELECT inserted_at FROM "+table+" WHERE log_id = 'new'").Scan(&insertedAt))
	require.GreaterOrEqual(t, insertedAt, now)
	var rows uint64
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT sum(rows) FROM system.projection_parts WHERE database = 'default' AND table = ? AND name = 'proj_logdrain' AND active`, strings.TrimPrefix(table, "default.")).Scan(&rows))
	require.EqualValues(t, 2, rows)
}
