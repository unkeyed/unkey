package clickhouse_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestConfigureUser_ProtectsPerQueryLimits guarantees workspace users retain
// read access but cannot disable or raise protected limits through raw SQL.
func TestConfigureUser_ProtectsPerQueryLimits(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := clickhouse.New(clickhouse.Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	password := "sqlsec_password"
	err = admin.ConfigureUser(ctx, clickhouse.UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       100,
		MaxExecutionTimePerWindow: 3600,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       64 * 1024 * 1024,
		MaxQueryResultRows:        100,
		RetentionDays:             30,
	})
	require.NoError(t, err)

	options, err := driver.ParseDSN(clickhouseConfig.DSN)
	require.NoError(t, err)
	options.Auth.Username = workspaceID
	options.Auth.Password = password

	workspaceConn, err := driver.Open(options)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workspaceConn.Close()) })

	var readonly uint8
	err = workspaceConn.QueryRow(ctx, "SELECT toUInt8(value) FROM system.settings WHERE name = 'readonly'").Scan(&readonly)
	require.NoError(t, err)
	require.Equal(t, uint8(1), readonly)

	// readonly=1 must prevent settings overrides without disabling ordinary
	// read queries for the workspace user.
	var result uint8
	err = workspaceConn.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	require.Equal(t, uint8(1), result)
	err = workspaceConn.QueryRow(ctx, "SELECT 1 SETTINGS max_execution_time = 29").Scan(&result)
	require.NoError(t, err)

	for _, value := range []int64{0, clickhouse.AnalyticsExecutionTimeMax + 1} {
		t.Run(fmt.Sprintf("max_execution_time/%d", value), func(t *testing.T) {
			err := workspaceConn.QueryRow(ctx, fmt.Sprintf("SELECT 1 SETTINGS max_execution_time = %d", value)).Scan(&result)
			var clickhouseErr *driver.Exception
			require.ErrorAs(t, err, &clickhouseErr)
			require.Equal(t, int32(452), clickhouseErr.Code)
		})
	}

	protectedSettings := []struct {
		name            string
		configuredValue int64
		readonly        uint8
		errorCode       int32
	}{
		{name: "max_execution_time", configuredValue: clickhouse.AnalyticsExecutionTimeMax, readonly: 0, errorCode: 452},
		{name: "max_memory_usage", configuredValue: 64 * 1024 * 1024, readonly: 1, errorCode: 164},
		{name: "max_result_rows", configuredValue: 100, readonly: 1, errorCode: 164},
	}
	for _, setting := range protectedSettings {
		var value uint64
		var isReadonly uint8
		err = workspaceConn.QueryRow(ctx,
			"SELECT toUInt64(value), readonly FROM system.settings WHERE name = ?",
			setting.name,
		).Scan(&value, &isReadonly)
		require.NoError(t, err)
		require.Equal(t, uint64(setting.configuredValue), value)
		require.Equal(t, setting.readonly, isReadonly)

		for _, value := range []int64{0, setting.configuredValue + 1} {
			t.Run(fmt.Sprintf("%s/%d", setting.name, value), func(t *testing.T) {
				err := workspaceConn.QueryRow(ctx, fmt.Sprintf("SELECT 1 SETTINGS %s = %d", setting.name, value)).Scan(&result)
				var clickhouseErr *driver.Exception
				require.ErrorAs(t, err, &clickhouseErr, "workspace user must not override %s", setting.name)
				require.Equal(t, setting.errorCode, clickhouseErr.Code)
			})
		}
	}
}

// TestConfigureUser_HidesInternalColumns proves the column grant on the
// gateway raw table is a real boundary rather than a convention. The query
// parser validates tables and functions but never columns, so ClickHouse is the
// only thing standing between a customer's SQL and the internal columns. Each
// subtest is a distinct way to reach a column without naming it in the SELECT
// list.
func TestConfigureUser_HidesInternalColumns(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := clickhouse.New(clickhouse.Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	password := "columns_password"

	err = admin.ConfigureUser(ctx, clickhouse.UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		AllowedTables:             clickhouse.DefaultAllowedTables(),
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       1000,
		MaxExecutionTimePerWindow: 3600,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       64 * 1024 * 1024,
		MaxQueryResultRows:        100,
		RetentionDays:             30,
	})
	require.NoError(t, err)

	// One row per workspace, so column visibility and row-level isolation can be
	// asserted against the same table.
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		workspaceID string
		path        string
	}{
		{workspaceID: workspaceID, path: "/kebap"},
		{workspaceID: otherWorkspaceID, path: "/other"},
	} {
		err = admin.Exec(ctx,
			"INSERT INTO default.frontline_requests_raw_v1 (request_id, time, workspace_id, project_id, app_id, environment_id, frontline_id, instance_address, platform, method, host, path, response_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			uid.New("req"), now, row.workspaceID, uid.New("proj"), uid.New("app"), uid.New("env"),
			"frontline_secret", "10.1.2.3", "k8s", "GET", "example.com", row.path, 200,
		)
		require.NoError(t, err)
	}

	workspaceURL, err := url.Parse(clickhouseConfig.HTTPDSN)
	require.NoError(t, err)
	workspaceURL.User = url.UserPassword(workspaceID, password)
	workspaceClient, err := clickhouse.New(clickhouse.Config{URL: workspaceURL.String()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workspaceClient.Close()) })

	internalColumns := []string{"instance_address", "frontline_id", "platform"}

	t.Run("granted columns stay readable", func(t *testing.T) {
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT path, response_status, total_latency FROM default.frontline_requests_raw_v1")
		require.NoError(t, err, "hidden columns must not break the rest of the table")
		require.Len(t, rows, 1)
		require.Equal(t, "/kebap", rows[0]["path"].(chcol.Variant).Any())
	})

	t.Run("row policy still isolates workspaces", func(t *testing.T) {
		// A column grant must not displace the row policy on the same table.
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT path FROM default.frontline_requests_raw_v1")
		require.NoError(t, err)
		require.Len(t, rows, 1, "the other workspace's row must stay invisible")
		require.Equal(t, "/kebap", rows[0]["path"].(chcol.Variant).Any())
	})

	t.Run("describe and show never name an internal column", func(t *testing.T) {
		// Introspection statements list every column of a table without selecting
		// from it, so the column list in a SELECT does not apply to them. The
		// analytics endpoint refuses non-SELECT statements before they get to
		// ClickHouse, but this asserts the layer below that check.
		introspection := []string{
			"DESCRIBE TABLE default.frontline_requests_raw_v1",
			"DESC default.frontline_requests_raw_v1",
			"SHOW CREATE TABLE default.frontline_requests_raw_v1",
			"SHOW COLUMNS FROM default.frontline_requests_raw_v1",
			"SELECT * FROM (DESCRIBE TABLE default.frontline_requests_raw_v1)",
		}

		for _, query := range introspection {
			t.Run(query, func(t *testing.T) {
				rows, err := workspaceClient.QueryToMaps(ctx, query)
				if err != nil {
					// A refusal is also a safe result. DESCRIBE, DESC, and SHOW
					// CREATE TABLE land here with ACCESS_DENIED, because the grant
					// gives SHOW COLUMNS only for the columns it names, not for the
					// table. SHOW COLUMNS is answered, but lists only those columns.
					return
				}
				serialized := fmt.Sprint(rows)
				for _, column := range internalColumns {
					require.NotContains(t, serialized, column,
						"%q disclosed the internal column %s", query, column)
				}
			})
		}
	})

	t.Run("schema introspection hides internal columns", func(t *testing.T) {
		// system.columns is filtered by grants rather than refused, so the probe
		// succeeds but must never name an internal column. The query parser blocks
		// system.* independently; this asserts the layer underneath it.
		for _, column := range internalColumns {
			rows, err := workspaceClient.QueryToMaps(ctx, fmt.Sprintf(
				"SELECT name FROM system.columns WHERE table = 'frontline_requests_raw_v1' AND name = '%s'", column))
			if err != nil {
				continue
			}
			require.Empty(t, rows, "system.columns disclosed the internal column %s", column)
		}
	})

	t.Run("select star never returns an internal column", func(t *testing.T) {
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT * FROM default.frontline_requests_raw_v1")
		if err != nil {
			// Refusing the wildcard outright is also a safe outcome.
			return
		}
		for _, row := range rows {
			for _, column := range internalColumns {
				require.NotContains(t, row, column, "SELECT * leaked an internal column")
			}
		}
	})

	for _, column := range internalColumns {
		t.Run("probing/"+column, func(t *testing.T) {
			probes := map[string]string{
				"direct select":      fmt.Sprintf("SELECT %s FROM default.frontline_requests_raw_v1", column),
				"where clause":       fmt.Sprintf("SELECT count() FROM default.frontline_requests_raw_v1 WHERE %s != ''", column),
				"aggregate":          fmt.Sprintf("SELECT max(%s) FROM default.frontline_requests_raw_v1", column),
				"group by":           fmt.Sprintf("SELECT count() FROM default.frontline_requests_raw_v1 GROUP BY %s", column),
				"order by":           fmt.Sprintf("SELECT path FROM default.frontline_requests_raw_v1 ORDER BY %s", column),
				"having":             fmt.Sprintf("SELECT path FROM default.frontline_requests_raw_v1 GROUP BY path HAVING max(%s) != ''", column),
				"aliased projection": fmt.Sprintf("SELECT %s AS leaked FROM default.frontline_requests_raw_v1", column),
				"subquery":           fmt.Sprintf("SELECT count() FROM (SELECT %s FROM default.frontline_requests_raw_v1)", column),
				"cte":                fmt.Sprintf("WITH probe AS (SELECT %s AS leaked FROM default.frontline_requests_raw_v1) SELECT count() FROM probe", column),
				"boolean oracle":     fmt.Sprintf("SELECT count() FROM default.frontline_requests_raw_v1 WHERE substring(%s, 1, 1) = 'k'", column),
				"union":              fmt.Sprintf("SELECT path FROM default.frontline_requests_raw_v1 UNION ALL SELECT %s FROM default.frontline_requests_raw_v1", column),
			}

			for name, query := range probes {
				t.Run(name, func(t *testing.T) {
					rows, err := workspaceClient.QueryToMaps(ctx, query)
					require.Error(t, err, "query must not reach %s: %s (returned %v)", column, query, rows)
				})
			}
		})
	}
}

// TestConfigureUser_HTTPTransport proves workspace users can query over HTTP
// with bounded API contexts.
func TestConfigureUser_HTTPTransport(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := clickhouse.New(clickhouse.Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	password := "http_password"
	err = admin.ConfigureUser(ctx, clickhouse.UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		AllowedTables:             []clickhouse.AllowedTable{{Name: "default.key_verifications_raw_v2", Columns: nil}},
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       100,
		MaxExecutionTimePerWindow: 3600,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       64 * 1024 * 1024,
		MaxQueryResultRows:        100,
		RetentionDays:             30,
	})
	require.NoError(t, err)

	t.Run("HTTP transport accepts the API deadline", func(t *testing.T) {
		// This locks the production client path. The server profile owns the
		// execution limit while the API context still owns cancellation.
		workspaceURL, err := url.Parse(clickhouseConfig.HTTPDSN)
		require.NoError(t, err)
		workspaceURL.User = url.UserPassword(workspaceID, password)

		workspaceClient, err := clickhouse.New(clickhouse.Config{URL: workspaceURL.String()})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, workspaceClient.Close()) })

		queryCtx, cancel := context.WithTimeout(ctx, clickhouse.AnalyticsQueryTimeout)
		defer cancel()
		rows, err := workspaceClient.QueryToMaps(queryCtx, "SELECT count() AS total FROM default.key_verifications_raw_v2")
		require.NoError(t, err, "a bounded API request must not conflict with the generated readonly profile")
		require.Len(t, rows, 1)

		t.Run("preserves context cancellation", func(t *testing.T) {
			queryCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			startedAt := time.Now()
			_, err := workspaceClient.QueryToMaps(queryCtx, "SELECT sleep(2)")
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Less(t, time.Since(startedAt), time.Second)
		})
	})
}
