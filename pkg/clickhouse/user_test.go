package clickhouse_test

import (
	"context"
	"fmt"
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

	protectedSettings := []struct {
		name            string
		configuredValue int64
	}{
		{name: "max_execution_time", configuredValue: 30},
		{name: "max_memory_usage", configuredValue: 64 * 1024 * 1024},
		{name: "max_result_rows", configuredValue: 100},
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
		require.Equal(t, uint8(1), isReadonly)

		for _, value := range []int64{0, setting.configuredValue + 1} {
			t.Run(fmt.Sprintf("%s/%d", setting.name, value), func(t *testing.T) {
				err := workspaceConn.QueryRow(ctx, fmt.Sprintf("SELECT 1 SETTINGS %s = %d", setting.name, value)).Scan(&result)
				var clickhouseErr *driver.Exception
				require.ErrorAs(t, err, &clickhouseErr, "workspace user must not override %s", setting.name)
				require.Equal(t, int32(164), clickhouseErr.Code)
			})
		}
	}
}
