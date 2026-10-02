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

func TestGatewayRequestProjection(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := t.Context()
	table := "default." + uid.New("gateway_projection")
	ddl, err := os.ReadFile("schema/022_frontline_requests_raw_v1.sql")
	require.NoError(t, err)
	require.NoError(t, client.conn.Exec(ctx, strings.Replace(string(ddl), "frontline_requests_raw_v1", table, 1)))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE "+table)) })
	const rowCount = 131072
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+`
		(workspace_id, request_id, time, inserted_at, project_id, response_status,
		app_id, environment_id, deployment_id, region, method, host, path,
		total_latency, instance_latency, gateway_latency, query_string, query_params,
		request_headers, request_body, response_headers, response_body, user_agent, ip_address)
		SELECT 'projection_workspace', leftPad(toString(number), 6, '0'),
		? - number, ? + number, toString(number % 32), if(number % 2 = 0, 201, 503),
		'app', 'env', 'deployment', 'eu-west-1', 'POST', 'api.example.com',
		concat('/orders/', toString(number)), 53, 41, 12, 'tag=a&tag=b', map('tag', ['a','b']),
		['Authorization: [REDACTED]'], repeat('request', 32), ['Content-Type: text/plain'], repeat('response', 32),
		'test-agent', '192.0.2.1' FROM numbers(?)`, now, now, rowCount))
	query := `SELECT inserted_at, time, request_id, project_id, app_id,
		environment_id, deployment_id, region, method, host, path, response_status,
		total_latency, instance_latency, gateway_latency, query_string, query_params,
		request_headers, request_body, response_headers, response_body, user_agent, ip_address
		FROM ` + table + ` WHERE workspace_id = 'projection_workspace'
		AND (inserted_at > {from_time:Int64} OR (inserted_at = {from_time:Int64} AND request_id > '130000'))
		AND inserted_at < {to:Int64}
		AND (empty({statuses:Array(Int32)}) OR intDiv(response_status, 100) IN {statuses:Array(Int32)})
		AND (empty({projects:Array(String)}) OR project_id IN {projects:Array(String)})
		AND (empty({apps:Array(String)}) OR app_id IN {apps:Array(String)})
		AND (empty({environments:Array(String)}) OR environment_id IN {environments:Array(String)})
		ORDER BY inserted_at, request_id LIMIT 1000
		SETTINGS min_table_rows_to_use_projection_index = 0`
	for _, tt := range []struct {
		name, statuses, projects, apps, environments string
		wantRows                                     int
	}{
		{name: "all statuses", statuses: "[]", projects: "[]", apps: "[]", environments: "[]", wantRows: 1000},
		{name: "2xx", statuses: "[2]", projects: "[]", apps: "[]", environments: "[]", wantRows: 535},
		{name: "5xx", statuses: "[5]", projects: "[]", apps: "[]", environments: "[]", wantRows: 536},
		{name: "combined filters", statuses: "[2]", projects: "['16']", apps: "['app']", environments: "['env']", wantRows: 33},
	} {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]string{"from_time": strconv.FormatInt(now+130000, 10), "to": strconv.FormatInt(now+rowCount, 10), "statuses": tt.statuses, "projects": tt.projects, "apps": tt.apps, "environments": tt.environments}
			queryCtx := ch.Context(ctx, ch.WithParameters(params))
			var plan []struct {
				Explain string `ch:"explain"`
			}
			require.NoError(t, client.conn.Select(queryCtx, &plan, "EXPLAIN projections=1, indexes=1 "+query))
			var explanation strings.Builder
			for _, line := range plan {
				explanation.WriteString(strings.TrimSpace(line.Explain) + "\n")
			}
			require.Contains(t, explanation.String(), "Name: proj_logdrain\nDescription: Projection has been analyzed and will be applied during reading")
			readRows := func(enabled bool) uint64 {
				t.Helper()
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
				var read uint64
				require.NoError(t, client.conn.QueryRow(ctx, `SELECT read_rows FROM system.query_log WHERE query_id = ? AND type = 'QueryFinish'`, id).Scan(&read))
				return read
			}
			without, with := readRows(false), readRows(true)
			t.Logf("diagnostic min_table_rows_to_use_projection_index=0; actual read_rows: off=%d on=%d", without, with)
			require.Positive(t, with)
			require.Positive(t, without)
		})
	}
}

func TestGatewayRequestMigration_PreservesInsertionTimestamps(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := t.Context()
	table := "default." + uid.New("gateway_migration")
	require.NoError(t, client.conn.Exec(ctx, `CREATE TABLE `+table+`
		(request_id String, time Int64, inserted_at Int64 DEFAULT toUnixTimestamp64Milli(now64(3)),
		workspace_id String, project_id String, app_id String, environment_id String, deployment_id String)
		ENGINE = MergeTree ORDER BY (workspace_id, project_id, app_id, environment_id, time, deployment_id)
		TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 7 DAY DELETE`))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(context.Background(), "DROP TABLE "+table)) })
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+` (workspace_id, request_id, time, inserted_at) VALUES ('workspace', 'old', ?, 12345)`, now))
	migration, err := os.ReadFile("migrations/20260910010000.sql")
	require.NoError(t, err)
	for _, statement := range strings.Split(string(migration), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		require.NoError(t, client.conn.Exec(ctx, strings.ReplaceAll(statement, "default.frontline_requests_raw_v1", table)))
	}
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+` (workspace_id, request_id, time) VALUES ('workspace', 'new', ?)`, now-60000))
	var historical, inserted int64
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT inserted_at FROM `+table+` WHERE request_id = 'old'`).Scan(&historical))
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT inserted_at FROM `+table+` WHERE request_id = 'new'`).Scan(&inserted))
	require.EqualValues(t, 12345, historical)
	require.GreaterOrEqual(t, inserted, now)
	var rows uint64
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT sum(rows) FROM system.projection_parts WHERE database = 'default' AND table = ? AND name = 'proj_logdrain' AND active`, strings.TrimPrefix(table, "default.")).Scan(&rows))
	require.EqualValues(t, 2, rows)
}
