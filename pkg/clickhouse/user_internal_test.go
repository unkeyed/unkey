package clickhouse

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

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

	// The rollups carry no infrastructure columns, so they stay unrestricted.
	for _, name := range []string{
		"default.frontline_requests_per_minute_v1",
		"default.frontline_requests_per_hour_v1",
		"default.frontline_requests_per_day_v1",
	} {
		table, found := byName[name]
		require.True(t, found, "%s must be granted", name)
		require.Empty(t, table.Columns, "%s must not be column-restricted", name)
	}

	// Rollups without a public alias must not be granted at all.
	require.NotContains(t, byName, "default.frontline_requests_per_5m_v1")
	require.NotContains(t, byName, "default.frontline_requests_per_15m_v1")

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

// TestGetTimeRetentionFilter_GatewayTables guarantees each gateway table lands
// in the branch matching its time column type. The raw table stores unix
// milliseconds while every rollup stores DateTime, so a misrouted table would
// produce a retention filter that never matches.
func TestGetTimeRetentionFilter_GatewayTables(t *testing.T) {
	tests := []struct {
		table    string
		expected string
	}{
		{
			table:    "default.frontline_requests_raw_v1",
			expected: "time >= toUnixTimestamp(toStartOfDay(now() - INTERVAL 30 DAY)) * 1000",
		},
		{
			table:    "default.frontline_requests_per_minute_v1",
			expected: "time >= toStartOfDay(now() - INTERVAL 30 DAY)",
		},
		{
			table:    "default.frontline_requests_per_hour_v1",
			expected: "time >= toStartOfDay(now() - INTERVAL 30 DAY)",
		},
		{
			// The day rollup stores DateTime rather than Date, unlike the
			// verification and ratelimit day tables this branch was written for.
			// ClickHouse promotes the Date bound to midnight DateTime, so the
			// comparison still holds.
			table:    "default.frontline_requests_per_day_v1",
			expected: "time >= today() - INTERVAL 30 DAY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			require.Equal(t, tt.expected, getTimeRetentionFilter(tt.table, 30))
		})
	}
}
