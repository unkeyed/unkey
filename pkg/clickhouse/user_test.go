package clickhouse

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestConfigureUser_ProtectsPerQueryLimits guarantees workspace users retain
// read access but cannot disable or raise protected limits through raw SQL.
func TestConfigureUser_ProtectsPerQueryLimits(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := New(Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	password := "sqlsec_password"
	err = admin.ConfigureUser(ctx, UserConfig{
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

	for _, value := range []int64{0, AnalyticsExecutionTimeMax + 1} {
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
		{name: "max_execution_time", configuredValue: AnalyticsExecutionTimeMax, readonly: 0, errorCode: 452},
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

	admin, err := New(Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	password := "columns_password"

	err = admin.ConfigureUser(ctx, UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		AllowedTables:             DefaultAllowedTables(),
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
	workspaceClient, err := New(Config{URL: workspaceURL.String()})
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

	admin, err := New(Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	password := "http_password"
	err = admin.ConfigureUser(ctx, UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		AllowedTables:             []AllowedTable{{Name: "default.key_verifications_raw_v2", Columns: nil}},
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

		workspaceClient, err := New(Config{URL: workspaceURL.String()})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, workspaceClient.Close()) })

		queryCtx, cancel := context.WithTimeout(ctx, AnalyticsQueryTimeout)
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

// TestGrantSelectSQL_RestrictsColumns guarantees a table carrying a column list
// produces a column-scoped grant. Column grants are the only enforcement point
// for column visibility, so a whole-table grant here would silently expose
// columns the endpoint never means to publish.
func TestGrantSelectSQL_RestrictsColumns(t *testing.T) {
	username := uid.New(uid.WorkspacePrefix)

	tests := []struct {
		name     string
		table    AllowedTable
		expected string
	}{
		{
			name:     "no columns grants the whole table",
			table:    AllowedTable{Name: "default.key_verifications_raw_v2", Columns: nil},
			expected: fmt.Sprintf("GRANT SELECT ON default.key_verifications_raw_v2 TO %s", username),
		},
		{
			name:     "empty column slice grants the whole table",
			table:    AllowedTable{Name: "default.ratelimits_raw_v2", Columns: []string{}},
			expected: fmt.Sprintf("GRANT SELECT ON default.ratelimits_raw_v2 TO %s", username),
		},
		{
			name:     "single column",
			table:    AllowedTable{Name: "default.frontline_requests_raw_v1", Columns: []string{"path"}},
			expected: fmt.Sprintf("GRANT SELECT(path) ON default.frontline_requests_raw_v1 TO %s", username),
		},
		{
			name:     "several columns",
			table:    AllowedTable{Name: "default.frontline_requests_raw_v1", Columns: []string{"time", "path", "response_status"}},
			expected: fmt.Sprintf("GRANT SELECT(time, path, response_status) ON default.frontline_requests_raw_v1 TO %s", username),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, grantSelectSQL(tt.table, username))
		})
	}
}

// TestDefaultAllowedTables_HidesGatewayInternalColumns guarantees the columns that
// describe Unkey's own infrastructure never reach a workspace user.
func TestDefaultAllowedTables_HidesGatewayInternalColumns(t *testing.T) {
	tables := DefaultAllowedTables()

	byName := make(map[string]AllowedTable, len(tables))
	for _, table := range tables {
		byName[table.Name] = table
	}

	raw, ok := byName["default.frontline_requests_raw_v1"]
	require.True(t, ok, "gateway raw table must be granted")
	require.NotEmpty(t, raw.Columns, "gateway raw table must carry a column allow-list")

	for _, internalColumn := range []string{"instance_address", "frontline_id", "platform"} {
		require.NotContains(t, raw.Columns, internalColumn)
	}
	for _, granted := range []string{"path", "response_status", "total_latency", "ip_address", "request_body"} {
		require.Contains(t, raw.Columns, granted)
	}

	// Converting the existing entries must not have narrowed them.
	for _, name := range []string{
		"default.key_verifications_raw_v2",
		"default.ratelimits_raw_v2",
	} {
		table, found := byName[name]
		require.True(t, found, "%s must still be granted", name)
		require.Empty(t, table.Columns)
	}
}

// TestDefaultAllowedTables_HidesRuntimeLogInternalColumns makes sure that the
// runtime logs grant gives access only to the documented columns. The query
// parser does not examine columns. Thus this grant is the only control on
// column access.
//
// attributes is the most important exclusion. The JSON type makes approximately
// one thousand subcolumn files for each part. attributes_text contains the same
// data, and a query reads it at a much lower cost.
func TestDefaultAllowedTables_HidesRuntimeLogInternalColumns(t *testing.T) {
	byName := make(map[string]AllowedTable)
	for _, table := range DefaultAllowedTables() {
		byName[table.Name] = table
	}

	raw, ok := byName["default.runtime_logs_raw_v1"]
	require.True(t, ok, "runtime logs table must be granted")
	require.NotEmpty(t, raw.Columns, "runtime logs table must carry a column allow-list")

	// k8s_pod_name is infrastructure data, not customer data. The deployment
	// endpoints make sure that k8s_name does not reach a response body. This
	// grant applies the same rule to this table.
	for _, internalColumn := range []string{"platform", "k8s_pod_name", "attributes", "expires_at"} {
		require.NotContains(t, raw.Columns, internalColumn)
	}
	for _, granted := range []string{
		"log_id", "time", "inserted_at", "severity", "message",
		"deployment_id", "region", "attributes_text",
	} {
		require.Contains(t, raw.Columns, granted)
	}

	// The row policy and the filter that the parser adds both use workspace_id.
	// Thus this column must stay readable.
	require.Contains(t, raw.Columns, "workspace_id")

	// The documentation tells customers to use these columns to select the data
	// of one project, app, or environment.
	for _, scope := range []string{"project_id", "app_id", "environment_id"} {
		require.Contains(t, raw.Columns, scope)
	}
}

// TestValidateIdentifiers_RejectsUnsafeColumns guarantees a malformed column
// never reaches the interpolated GRANT statement. ClickHouse cannot
// parameterize identifiers, so this validation is the injection guard.
func TestValidateIdentifiers_RejectsUnsafeColumns(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)

	unsafe := []string{
		"path, instance_address",
		"path) ON default.frontline_requests_raw_v1 TO other (",
		"pa'th",
		"pa`th",
		"pa th",
		"path;",
		"*",
		"",
	}

	for _, column := range unsafe {
		t.Run(fmt.Sprintf("column/%q", column), func(t *testing.T) {
			err := validateIdentifiers(UserConfig{
				Username:      workspaceID,
				WorkspaceID:   workspaceID,
				AllowedTables: []AllowedTable{{Name: "default.frontline_requests_raw_v1", Columns: []string{column}}},
			})
			require.Error(t, err)
		})
	}

	err := validateIdentifiers(UserConfig{
		Username:      workspaceID,
		WorkspaceID:   workspaceID,
		AllowedTables: DefaultAllowedTables(),
	})
	require.NoError(t, err, "the shipped table list must pass its own validation")
}

// TestGetTimeRetentionFilter_GatewayTables guarantees the gateway raw table
// lands in the unix-milliseconds branch. A misrouted table would produce a
// retention filter that never matches.
func TestGetTimeRetentionFilter_GatewayTables(t *testing.T) {
	require.Equal(t,
		"time >= toUnixTimestamp(toStartOfDay(now() - INTERVAL 30 DAY)) * 1000",
		getTimeRetentionFilter("default.frontline_requests_raw_v1", 30),
	)
}

// TestGetTimeRetentionFilter_RuntimeLogsTable makes sure that the runtime logs
// table uses the branch for raw tables. Its time column is an Int64 that
// contains unix milliseconds. A DateTime filter would compare milliseconds with
// seconds. The query would then return no rows and give no error.
func TestGetTimeRetentionFilter_RuntimeLogsTable(t *testing.T) {
	require.Equal(t,
		"time >= toUnixTimestamp(toStartOfDay(now() - INTERVAL 30 DAY)) * 1000",
		getTimeRetentionFilter("default.runtime_logs_raw_v1", 30),
	)
}

// TestConfigureUser_HidesRuntimeLogInternalColumns shows that the column grant
// on the runtime logs table is a real boundary. It does the same as
// TestConfigureUser_HidesInternalColumns does for the gateway table.
//
// It also tests the condition that the grant depends on. ClickHouse computes
// attributes_text from attributes with MATERIALIZED. This test makes sure that
// ClickHouse gives access to the materialized column but refuses the JSON
// column. If ClickHouse refused both columns, the endpoint would have to grant
// the JSON column and accept its higher read cost.
func TestConfigureUser_HidesRuntimeLogInternalColumns(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := New(Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	password := "runtime_logs_password"

	err = admin.ConfigureUser(ctx, UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		AllowedTables:             DefaultAllowedTables(),
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       1000,
		MaxExecutionTimePerWindow: 3600,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       64 * 1024 * 1024,
		MaxQueryResultRows:        100,
		RetentionDays:             30,
	})
	require.NoError(t, err)

	// There is one row for each workspace. Thus one table shows both the column
	// visibility and the row isolation. Each internal column has a value that is
	// not empty. If a column were empty, a probe could get an empty result and
	// look successful when ClickHouse did not refuse it.
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		workspaceID string
		message     string
	}{
		{workspaceID: workspaceID, message: "kebap served"},
		{workspaceID: otherWorkspaceID, message: "other workspace log"},
	} {
		err = admin.Exec(ctx,
			"INSERT INTO default.runtime_logs_raw_v1 (log_id, time, inserted_at, severity, message, workspace_id, project_id, environment_id, app_id, deployment_id, k8s_pod_name, region, platform, attributes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			uid.New("log"), now, now, "info", row.message, row.workspaceID,
			uid.New("proj"), uid.New("env"), uid.New("app"), uid.New("dep"),
			"pod-abc-123", "local", "k8s", `{"user_id":"usr_kebap"}`,
		)
		require.NoError(t, err)
	}

	workspaceURL, err := url.Parse(clickhouseConfig.HTTPDSN)
	require.NoError(t, err)
	workspaceURL.User = url.UserPassword(workspaceID, password)
	workspaceClient, err := New(Config{URL: workspaceURL.String()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workspaceClient.Close()) })

	internalColumns := []string{"platform", "k8s_pod_name", "attributes", "expires_at"}

	t.Run("granted columns stay readable", func(t *testing.T) {
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT severity, message, deployment_id FROM default.runtime_logs_raw_v1")
		require.NoError(t, err, "hidden columns must not break the rest of the table")
		require.Len(t, rows, 1)
		require.Equal(t, "kebap served", rows[0]["message"].(chcol.Variant).Any())
	})

	t.Run("attributes_text is readable while attributes is not", func(t *testing.T) {
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT attributes_text FROM default.runtime_logs_raw_v1")
		require.NoError(t, err, "the materialized column must be readable on its own")
		require.Len(t, rows, 1)
		require.Contains(t, fmt.Sprint(rows[0]["attributes_text"].(chcol.Variant).Any()), "usr_kebap")

		_, err = workspaceClient.QueryToMaps(ctx,
			"SELECT attributes FROM default.runtime_logs_raw_v1")
		require.Error(t, err, "the JSON column must stay unreachable")
	})

	t.Run("row policy still isolates workspaces", func(t *testing.T) {
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT message FROM default.runtime_logs_raw_v1")
		require.NoError(t, err)
		require.Len(t, rows, 1, "the other workspace's row must stay invisible")
		require.Equal(t, "kebap served", rows[0]["message"].(chcol.Variant).Any())
	})

	t.Run("schema introspection hides internal columns", func(t *testing.T) {
		for _, column := range internalColumns {
			rows, err := workspaceClient.QueryToMaps(ctx, fmt.Sprintf(
				"SELECT name FROM system.columns WHERE table = 'runtime_logs_raw_v1' AND name = '%s'", column))
			if err != nil {
				continue
			}
			require.Empty(t, rows, "system.columns disclosed the internal column %s", column)
		}
	})

	t.Run("select star never returns an internal column", func(t *testing.T) {
		rows, err := workspaceClient.QueryToMaps(ctx,
			"SELECT * FROM default.runtime_logs_raw_v1")
		if err != nil {
			// A refusal of the wildcard is also a safe result.
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
				"direct select":      fmt.Sprintf("SELECT %s FROM default.runtime_logs_raw_v1", column),
				"where clause":       fmt.Sprintf("SELECT count() FROM default.runtime_logs_raw_v1 WHERE toString(%s) != ''", column),
				"aggregate":          fmt.Sprintf("SELECT max(toString(%s)) FROM default.runtime_logs_raw_v1", column),
				"group by":           fmt.Sprintf("SELECT count() FROM default.runtime_logs_raw_v1 GROUP BY %s", column),
				"order by":           fmt.Sprintf("SELECT message FROM default.runtime_logs_raw_v1 ORDER BY %s", column),
				"aliased projection": fmt.Sprintf("SELECT %s AS leaked FROM default.runtime_logs_raw_v1", column),
				"subquery":           fmt.Sprintf("SELECT count() FROM (SELECT %s FROM default.runtime_logs_raw_v1)", column),
				"cte":                fmt.Sprintf("WITH probe AS (SELECT %s AS leaked FROM default.runtime_logs_raw_v1) SELECT count() FROM probe", column),
				"boolean oracle":     fmt.Sprintf("SELECT count() FROM default.runtime_logs_raw_v1 WHERE substring(toString(%s), 1, 1) = 'k'", column),
				"union":              fmt.Sprintf("SELECT message FROM default.runtime_logs_raw_v1 UNION ALL SELECT toString(%s) FROM default.runtime_logs_raw_v1", column),
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
