package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestGatewayParserConfigRequiresPublicAliases guarantees the endpoint accepts
// only the four public aliases. Physical table names must stay unusable so that
// renaming a frontline table never becomes a breaking API change, and so that
// the ungranted per_5m and per_15m rollups stay out of reach.
func TestGatewayParserConfigRequiresPublicAliases(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)

	newParser := func() *queryparser.Parser {
		return queryparser.NewParser(queryparser.Config{
			WorkspaceID:       workspaceID,
			TableAliases:      tableAliases,
			AllowedTables:     allowedTables,
			SecurityFilters:   nil,
			Limit:             0,
			QueryRangeDaysMax: 0,
		})
	}

	t.Run("every public alias resolves", func(t *testing.T) {
		for alias, physical := range tableAliases {
			t.Run(alias, func(t *testing.T) {
				parsed, err := newParser().Parse(context.Background(), "SELECT count() FROM "+alias)
				require.NoError(t, err)
				require.Contains(t, parsed, physical, "the alias must rewrite to its physical table")
				require.Contains(t, parsed, workspaceID, "the workspace filter must be injected")
			})
		}
	})

	// Each query below appears in the OpenAPI spec or in the docs page for this
	// endpoint. A documented query that the parser rejects is a documentation
	// bug, thus they are asserted here and not only reviewed by eye.
	t.Run("documented queries parse", func(t *testing.T) {
		documented := map[string]string{
			"docs: recent requests":      "SELECT path, response_status, total_latency FROM gateway_requests_v1 ORDER BY time DESC LIMIT 10",
			"docs: latency percentiles":  "SELECT quantileTDigestMerge(0.5)(latency_p50) AS p50, quantileTDigestMerge(0.95)(latency_p95) AS p95, quantileTDigestMerge(0.99)(latency_p99) AS p99 FROM gateway_requests_per_hour_v1 WHERE app_id = 'app_1234' AND time >= now() - INTERVAL 24 HOUR",
			"docs: raw latency quantile": "SELECT quantile(0.95)(total_latency) AS p95 FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 1 HOUR)",
			"docs: paths with errors":    "SELECT path, count() AS total FROM gateway_requests_v1 WHERE response_status >= 500 AND time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 24 HOUR) GROUP BY path ORDER BY total DESC LIMIT 10",
			"docs: error rate":           "SELECT deployment_id, count() AS total, countIf(response_status >= 500) AS errors, round(countIf(response_status >= 500) / count() * 100, 2) AS error_rate FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 24 HOUR) GROUP BY deployment_id ORDER BY error_rate DESC",
			"docs: zero filled series":   "SELECT time, sum(count) AS requests FROM gateway_requests_per_minute_v1 WHERE app_id = 'app_1234' AND time >= toStartOfMinute(now() - INTERVAL 6 HOUR) AND time < toStartOfMinute(now()) GROUP BY time ORDER BY time WITH FILL FROM toStartOfMinute(now() - INTERVAL 6 HOUR) TO toStartOfMinute(now()) STEP INTERVAL 1 MINUTE",
			"docs: status per app":       "SELECT app_id, response_status, sum(count) AS total FROM gateway_requests_per_hour_v1 WHERE time >= now() - INTERVAL 7 DAY GROUP BY app_id, response_status ORDER BY app_id, total DESC",
			"docs: environment filter":   "SELECT count() FROM gateway_requests_v1 WHERE environment_id IN ('env_1234', 'env_5678')",
			"request body example":       "SELECT path, count() AS total FROM gateway_requests_v1 WHERE response_status >= 500 AND time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 24 HOUR) GROUP BY path ORDER BY total DESC LIMIT 10",
			"latency merge state":        "SELECT quantileTDigestMerge(0.95)(latency_p95) AS p95 FROM gateway_requests_per_hour_v1",
			"cte":                        "WITH totals AS (SELECT count() AS total FROM gateway_requests_v1) SELECT total FROM totals",
			"subquery":                   "SELECT sum(total) FROM (SELECT count() AS total FROM gateway_requests_v1)",
			"union":                      "SELECT count() FROM gateway_requests_v1 UNION ALL SELECT count() FROM gateway_requests_per_hour_v1",
			"except":                     "SELECT path FROM gateway_requests_v1 EXCEPT SELECT path FROM gateway_requests_v1",
			"scope filter":               "SELECT count() FROM gateway_requests_v1 WHERE project_id = 'proj_123' AND app_id = 'app_123' AND environment_id = 'env_123'",
		}

		for name, query := range documented {
			t.Run(name, func(t *testing.T) {
				_, err := newParser().Parse(context.Background(), query)
				require.NoError(t, err, "the spec documents this query, so it must parse")
			})
		}
	})

	rejected := map[string]string{
		"physical raw table":     "SELECT count() FROM default.frontline_requests_raw_v1",
		"unqualified raw table":  "SELECT count() FROM frontline_requests_raw_v1",
		"physical hour rollup":   "SELECT count() FROM default.frontline_requests_per_hour_v1",
		"ungranted 5m rollup":    "SELECT count() FROM gateway_requests_per_5m_v1",
		"ungranted 15m rollup":   "SELECT count() FROM gateway_requests_per_15m_v1",
		"another endpoint table": "SELECT count() FROM key_verifications_v1",
		"physical name in a CTE": "WITH probe AS (SELECT count() AS total FROM default.frontline_requests_raw_v1) SELECT total FROM probe",
		"physical name in union": "SELECT count() FROM gateway_requests_v1 UNION ALL SELECT count() FROM default.frontline_requests_raw_v1",
		"system table":           "SELECT count() FROM system.tables",
	}

	for name, query := range rejected {
		t.Run(name, func(t *testing.T) {
			_, err := newParser().Parse(context.Background(), query)
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok, "the rejection must carry an error code")
			require.Equal(t, codes.User.BadRequest.InvalidAnalyticsTable.URN(), code)
		})
	}
}
