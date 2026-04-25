package urql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
)

func compile(t *testing.T, query string) (string, map[string]string, error) {
	t.Helper()
	return Compile(context.Background(), query, DefaultSchema)
}

func TestCompile_TableResolution_Raw(t *testing.T) {
	// No timeBucket() → raw variant.
	sql, formats, err := compile(t, "SELECT outcome FROM key_verifications WHERE time > now() - INTERVAL 1 DAY")
	require.NoError(t, err)
	require.Nil(t, formats)
	require.Contains(t, sql, "default.key_verifications_raw_v2")
}

func TestCompile_TableResolution_PerMinute(t *testing.T) {
	// timeBucket() + 6h range → per_minute variant.
	sql, _, err := compile(t, "SELECT timeBucket(), count() FROM key_verifications WHERE time > now() - INTERVAL 1 HOUR GROUP BY timeBucket")
	require.NoError(t, err)
	require.Contains(t, sql, "default.key_verifications_per_minute_v3")
}

func TestCompile_TableResolution_PerHour(t *testing.T) {
	// timeBucket() + 7d range → per_hour variant.
	sql, _, err := compile(t, "SELECT timeBucket(), count() FROM key_verifications WHERE time > now() - INTERVAL 7 DAY GROUP BY timeBucket")
	require.NoError(t, err)
	require.Contains(t, sql, "default.key_verifications_per_hour_v3")
}

func TestCompile_TableResolution_PerDay(t *testing.T) {
	sql, _, err := compile(t, "SELECT timeBucket(), count() FROM key_verifications WHERE time > now() - INTERVAL 30 DAY GROUP BY timeBucket")
	require.NoError(t, err)
	require.Contains(t, sql, "default.key_verifications_per_day_v3")
}

func TestCompile_TableResolution_PerMonth(t *testing.T) {
	sql, _, err := compile(t, "SELECT timeBucket(), count() FROM key_verifications WHERE time > now() - INTERVAL 1 YEAR GROUP BY timeBucket")
	require.NoError(t, err)
	require.Contains(t, sql, "default.key_verifications_per_month_v3")
}

func TestCompile_TimeBucketReplacement(t *testing.T) {
	sql, _, err := compile(t, "SELECT timeBucket(), count() FROM key_verifications WHERE time > now() - INTERVAL 1 DAY GROUP BY timeBucket")
	require.NoError(t, err)
	require.NotContains(t, sql, "timeBucket")
	require.Contains(t, sql, " time")
}

func TestCompile_VirtualColumn_IsSuccessful(t *testing.T) {
	sql, _, err := compile(t, "SELECT is_successful FROM key_verifications WHERE time > now() - INTERVAL 1 DAY")
	require.NoError(t, err)
	require.NotContains(t, sql, "is_successful")
	require.Contains(t, sql, "outcome = 'VALID'")
}

func TestCompile_VirtualColumn_InWhere(t *testing.T) {
	// Virtual columns also expand inside WHERE expressions.
	sql, _, err := compile(t, "SELECT outcome FROM key_verifications WHERE is_successful AND time > now() - INTERVAL 1 DAY")
	require.NoError(t, err)
	require.NotContains(t, sql, "is_successful")
	require.Contains(t, sql, "outcome = 'VALID'")
}

func TestCompile_AllowedValues_Pass(t *testing.T) {
	_, _, err := compile(t, "SELECT outcome FROM key_verifications WHERE outcome = 'VALID' AND time > now() - INTERVAL 1 DAY")
	require.NoError(t, err)
}

func TestCompile_AllowedValues_Reject(t *testing.T) {
	_, _, err := compile(t, "SELECT outcome FROM key_verifications WHERE outcome = 'NOPE' AND time > now() - INTERVAL 1 DAY")
	require.Error(t, err)
	require.Contains(t, err.Error(), "NOPE")
	require.Contains(t, err.Error(), "outcome")
}

func TestCompile_AllowedValues_RejectInList(t *testing.T) {
	_, _, err := compile(t, "SELECT outcome FROM key_verifications WHERE outcome IN ('VALID', 'NOPE') AND time > now() - INTERVAL 1 DAY")
	require.Error(t, err)
	require.Contains(t, err.Error(), "NOPE")
}

func TestCompile_AllowedValues_RejectInHaving(t *testing.T) {
	// HAVING is also walked for allowedValues.
	_, _, err := compile(t, "SELECT outcome FROM key_verifications WHERE time > now() - INTERVAL 1 DAY GROUP BY outcome HAVING outcome = 'NOPE'")
	require.Error(t, err)
	require.Contains(t, err.Error(), "NOPE")
}

func TestCompile_PrettyFormat_Aliased(t *testing.T) {
	sql, formats, err := compile(t, "SELECT timeBucket(), prettyFormat(count(), 'quantity') AS total FROM key_verifications WHERE time > now() - INTERVAL 1 DAY GROUP BY timeBucket")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"total": "quantity"}, formats)
	require.NotContains(t, sql, "prettyFormat")
	require.Contains(t, sql, "count()")
}

func TestCompile_PrettyFormat_UnknownFormat(t *testing.T) {
	_, _, err := compile(t, "SELECT prettyFormat(count(), 'meters') AS x FROM key_verifications WHERE time > now() - INTERVAL 1 DAY")
	require.Error(t, err)
	require.Contains(t, err.Error(), "meters")
}

func TestCompile_PrettyFormat_RequiresAlias(t *testing.T) {
	_, _, err := compile(t, "SELECT prettyFormat(count(), 'quantity') FROM key_verifications WHERE time > now() - INTERVAL 1 DAY")
	require.Error(t, err)
	require.Contains(t, err.Error(), "alias")
}

func TestCompile_TimeBucket_RejectsArgs(t *testing.T) {
	_, _, err := compile(t, "SELECT timeBucket('5 minutes') FROM key_verifications WHERE time > now() - INTERVAL 1 DAY GROUP BY timeBucket")
	require.Error(t, err)
	require.Contains(t, err.Error(), "argument")
}

func TestCompile_UnknownColumn(t *testing.T) {
	_, _, err := compile(t, "SELECT badcol FROM key_verifications WHERE time > now() - INTERVAL 1 DAY")
	require.Error(t, err)
	require.Contains(t, err.Error(), "badcol")
}

func TestCompile_RawOnlyColumn_OnAggregatedRejected(t *testing.T) {
	// `latency` exists only on raw. With timeBucket() forcing per_*, it's not available.
	_, _, err := compile(t, "SELECT timeBucket(), latency FROM key_verifications WHERE time > now() - INTERVAL 1 DAY GROUP BY timeBucket")
	require.Error(t, err)
	require.Contains(t, err.Error(), "latency")
}

func TestCompile_AggregatedOnlyColumn_OnRawRejected(t *testing.T) {
	// `count` is an aggregated-table column; on raw it doesn't exist.
	_, _, err := compile(t, "SELECT count FROM key_verifications WHERE time > now() - INTERVAL 1 DAY")
	require.Error(t, err)
}

func TestCompile_Fallback_UnknownTable(t *testing.T) {
	_, _, err := compile(t, "SELECT * FROM unknown_table")
	require.ErrorIs(t, err, ErrNotURQL)
}

func TestCompile_Fallback_LegacyAlias(t *testing.T) {
	_, _, err := compile(t, "SELECT * FROM key_verifications_v1")
	require.ErrorIs(t, err, ErrNotURQL)
}

func TestCompile_Fallback_PhysicalTable(t *testing.T) {
	_, _, err := compile(t, "SELECT * FROM default.key_verifications_raw_v2")
	require.ErrorIs(t, err, ErrNotURQL)
}

func TestCompile_Fallback_NonSelect(t *testing.T) {
	// AfterShip parses INSERT but URQL only owns SELECTs.
	_, _, err := compile(t, "INSERT INTO key_verifications VALUES (1)")
	require.ErrorIs(t, err, ErrNotURQL)
}

func TestCompile_Mixed_Rejected(t *testing.T) {
	// URQL logical table + legacy alias in same query → URQL owns + rejects.
	_, _, err := compile(t, "SELECT * FROM key_verifications JOIN default.something ON 1=1")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotURQL))
	require.Contains(t, err.Error(), "default.something")
}

func TestCompile_SelfJoin_SameVariant(t *testing.T) {
	// Both sides resolve to the same physical variant (raw, since no timeBucket).
	sql, _, err := compile(t, "SELECT a.outcome, b.outcome FROM key_verifications a JOIN key_verifications b ON a.request_id = b.request_id WHERE a.time > now() - INTERVAL 1 DAY")
	require.NoError(t, err)
	count := strings.Count(sql, "default.key_verifications_raw_v2")
	require.Equal(t, 2, count, "both join sides should resolve to the same physical variant; got SQL: %s", sql)
}

func TestCompile_CTE_InnerReference(t *testing.T) {
	// CTE body references key_verifications; outer references the CTE name.
	sql, _, err := compile(t, "WITH recent AS (SELECT * FROM key_verifications WHERE time > now() - INTERVAL 1 HOUR) SELECT count() FROM recent")
	require.NoError(t, err)
	require.Contains(t, sql, "default.key_verifications_raw_v2")
	require.Contains(t, sql, "recent")
}

func TestCompile_Subquery(t *testing.T) {
	sql, _, err := compile(t, "SELECT * FROM (SELECT outcome, count() c FROM key_verifications WHERE time > now() - INTERVAL 1 DAY GROUP BY outcome)")
	require.NoError(t, err)
	require.Contains(t, sql, "default.key_verifications_raw_v2")
}

// TestCompile_E2E_WithSecurityParser confirms URQL output is consumable by
// the existing query-parser, which is the production pipeline. Workspace
// isolation must still be injected on the URQL-rewritten query.
func TestCompile_E2E_WithSecurityParser(t *testing.T) {
	urqlSQL, _, err := compile(t, "SELECT outcome, count() FROM key_verifications WHERE time > now() - INTERVAL 1 HOUR GROUP BY outcome")
	require.NoError(t, err)

	parser := queryparser.NewParser(queryparser.Config{
		WorkspaceID: "ws_test",
		Limit:       1000,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
			"default.key_verifications_per_minute_v3",
			"default.key_verifications_per_hour_v3",
			"default.key_verifications_per_day_v3",
			"default.key_verifications_per_month_v3",
		},
		MaxQueryRangeDays: 90,
	})
	finalSQL, err := parser.Parse(context.Background(), urqlSQL)
	require.NoError(t, err)
	require.Contains(t, finalSQL, "workspace_id = 'ws_test'")
	require.Contains(t, finalSQL, "default.key_verifications_raw_v2")
}

func TestCompile_E2E_WithTimeBucket(t *testing.T) {
	urqlSQL, _, err := compile(t, "SELECT timeBucket(), count() FROM key_verifications WHERE time > now() - INTERVAL 7 DAY GROUP BY timeBucket")
	require.NoError(t, err)

	parser := queryparser.NewParser(queryparser.Config{
		WorkspaceID: "ws_test",
		Limit:       1000,
		AllowedTables: []string{
			"default.key_verifications_per_hour_v3",
		},
		MaxQueryRangeDays: 90,
	})
	finalSQL, err := parser.Parse(context.Background(), urqlSQL)
	require.NoError(t, err)
	require.Contains(t, finalSQL, "workspace_id = 'ws_test'")
	require.Contains(t, finalSQL, "default.key_verifications_per_hour_v3")
	require.NotContains(t, finalSQL, "timeBucket")
}
