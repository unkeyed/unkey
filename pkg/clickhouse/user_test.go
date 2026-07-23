package clickhouse_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// TestConfigureUser_ProtectsPerQueryLimits guarantees workspace users retain
// read access but cannot disable or raise protected limits through raw SQL.
func TestConfigureUser_ProtectsPerQueryLimits(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := clickhouse.New(clickhouse.Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := fmt.Sprintf("ws_sqlsec_%d", time.Now().UnixNano())
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

// TestConfigureUser_HTTPTransportAndMetadataContracts proves workspace users
// can query over HTTP with bounded API contexts and cannot inspect physical
// schema metadata.
func TestConfigureUser_HTTPTransportAndMetadataContracts(t *testing.T) {
	ctx := context.Background()
	clickhouseConfig := containers.ClickHouse(t)

	admin, err := clickhouse.New(clickhouse.Config{URL: clickhouseConfig.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	workspaceID := fmt.Sprintf("ws_httpmeta_%d", time.Now().UnixNano())
	password := "httpmeta_password"
	err = admin.ConfigureUser(ctx, clickhouse.UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  workspaceID,
		Password:                  password,
		AllowedTables:             []string{"default.key_verifications_raw_v2"},
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       100,
		MaxExecutionTimePerWindow: 3600,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       64 * 1024 * 1024,
		MaxQueryResultRows:        100,
		RetentionDays:             30,
	})
	require.NoError(t, err)

	t.Run("admin retains metadata visibility", func(t *testing.T) {
		// A workspace policy must not change metadata visibility for users that
		// are not assigned that policy.
		for _, query := range []string{
			"SELECT count() FROM system.tables WHERE database = 'default' AND name = 'key_verifications_raw_v2'",
			"SELECT count() FROM system.columns WHERE database = 'default' AND table = 'key_verifications_raw_v2'",
		} {
			var count uint64
			err := admin.Conn().QueryRow(ctx, query).Scan(&count)
			require.NoError(t, err)
			require.Positive(t, count)
		}
	})

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

	t.Run("direct user cannot inspect physical schema metadata", func(t *testing.T) {
		// Table-specific grants must not reveal the physical table and column
		// inventory through ClickHouse metadata tables.
		options, err := driver.ParseDSN(clickhouseConfig.DSN)
		require.NoError(t, err)
		options.Auth.Username = workspaceID
		options.Auth.Password = password

		workspaceConn, err := driver.Open(options)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, workspaceConn.Close()) })

		queries := []struct {
			name  string
			query string
		}{
			{
				name:  "system.tables",
				query: "SELECT count() FROM system.tables WHERE database = 'default' AND name = 'key_verifications_raw_v2'",
			},
			{
				name:  "system.columns",
				query: "SELECT count() FROM system.columns WHERE database = 'default' AND table = 'key_verifications_raw_v2'",
			},
		}
		for _, query := range queries {
			t.Run(query.name, func(t *testing.T) {
				var count uint64
				err := workspaceConn.QueryRow(ctx, query.query).Scan(&count)
				if err == nil {
					require.Zero(t, count, "workspace user must not see physical schema metadata through %s", query.name)
					return
				}

				var clickhouseErr *driver.Exception
				require.ErrorAs(t, err, &clickhouseErr, "workspace user must not read %s", query.name)
				require.Equal(t, int32(497), clickhouseErr.Code)
			})
		}
	})
}
