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

func TestKeyVerificationProjection(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	table := "default." + uid.New("verification_projection")
	ddl, err := os.ReadFile("schema/001_key_verifications_raw_v2.sql")
	require.NoError(t, err)
	require.NoError(t, client.conn.Exec(ctx, strings.Replace(string(ddl), "key_verifications_raw_v2", table, 1)))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(ctx, "DROP TABLE "+table)) })
	const rowCount = 131072
	now := time.Now().UnixMilli()
	// The base key orders event time in the opposite direction to ingestion.
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+`
		(workspace_id, request_id, time, inserted_at, outcome, key_space_id)
		SELECT 'projection_workspace', leftPad(toString(number), 6, '0'),
		? - number, ? + number,
		if(number % 2 = 0, 'VALID', 'RATE_LIMITED'),
		if(number % 3 = 0, 'selected', 'other') FROM numbers(?)`, now, now, rowCount))
	query := `SELECT inserted_at, time, request_id, key_space_id,
		identity_id, external_id, key_id, region, source, app_id, outcome, tags, spent_credits
		FROM ` + table + ` WHERE workspace_id = 'projection_workspace'
		AND (inserted_at > {from_time:Int64} OR (inserted_at = {from_time:Int64} AND request_id > '130000'))
		AND inserted_at < {to:Int64}
		AND (empty({outcomes:Array(String)}) OR outcome IN {outcomes:Array(String)})
		AND (empty({key_space_ids:Array(String)}) OR key_space_id IN {key_space_ids:Array(String)})
		ORDER BY inserted_at, request_id LIMIT 1000
		SETTINGS min_table_rows_to_use_projection_index = 0`
	for _, tt := range []struct {
		name      string
		outcomes  []string
		keySpaces []string
		wantRows  int
	}{
		{name: "all outcomes", wantRows: 1000},
		{name: "selected outcome", outcomes: []string{"VALID"}, wantRows: 535},
		{name: "selected keyspace", keySpaces: []string{"selected"}, wantRows: 357},
		{name: "selected keyspace and outcome", outcomes: []string{"VALID"}, keySpaces: []string{"selected"}, wantRows: 179},
	} {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]string{"from_time": strconv.FormatInt(now+130000, 10), "to": strconv.FormatInt(now+rowCount, 10), "outcomes": StringArrayParam(tt.outcomes), "key_space_ids": StringArrayParam(tt.keySpaces)}
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
				require.NoError(t, client.conn.QueryRow(context.Background(), `SELECT read_rows FROM system.query_log WHERE query_id = ? AND type = 'QueryFinish'`, id).Scan(&read))
				return read
			}
			without := readRows(false)
			with := readRows(true)
			t.Logf("actual read_rows: projection off=%d on=%d", without, with)
			require.Positive(t, with)
			require.Less(t, with, without)
		})
	}
}

func TestKeyVerificationMigration_PreservesHistory(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	table := "default." + uid.New("verification_migration")
	require.NoError(t, client.conn.Exec(ctx, `CREATE TABLE `+table+`
		(request_id String, time Int64, workspace_id String, key_space_id String, outcome String)
		ENGINE = MergeTree ORDER BY (workspace_id, time, key_space_id, outcome)`))
	t.Cleanup(func() { require.NoError(t, client.conn.Exec(ctx, "DROP TABLE "+table)) })
	now := time.Now().UnixMilli()
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+` (workspace_id, request_id, time) VALUES ('workspace', 'old', ?)`, now))
	migration, err := os.ReadFile("migrations/20260910000000.sql")
	require.NoError(t, err)
	for _, statement := range strings.Split(string(migration), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		require.NoError(t, client.conn.Exec(ctx, strings.ReplaceAll(statement, "default.key_verifications_raw_v2", table)))
	}
	require.NoError(t, client.conn.Exec(ctx, `INSERT INTO `+table+` (workspace_id, request_id, time) VALUES ('workspace', 'new', ?)`, now-60000))
	var historical, inserted int64
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT inserted_at FROM `+table+` WHERE request_id = 'old'`).Scan(&historical))
	require.NoError(t, client.conn.QueryRow(ctx, `SELECT inserted_at FROM `+table+` WHERE request_id = 'new'`).Scan(&inserted))
	require.Zero(t, historical)
	require.GreaterOrEqual(t, inserted, now)
}
