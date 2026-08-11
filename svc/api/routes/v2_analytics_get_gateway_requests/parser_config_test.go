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

	// Each query below appears in the OpenAPI spec for this endpoint. A claim in
	// the spec that the parser rejects is a documentation bug, so they are
	// asserted here and not only reviewed by eye.
	t.Run("documented queries parse", func(t *testing.T) {
		documented := map[string]string{
			"request body example": "SELECT path, count() AS total FROM gateway_requests_v1 WHERE response_status >= 500 AND time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 24 HOUR) GROUP BY path ORDER BY total DESC LIMIT 10",
			"latency merge state":  "SELECT quantileTDigestMerge(0.95)(latency_p95) AS p95 FROM gateway_requests_per_hour_v1",
			"cte":                  "WITH totals AS (SELECT count() AS total FROM gateway_requests_v1) SELECT total FROM totals",
			"subquery":             "SELECT sum(total) FROM (SELECT count() AS total FROM gateway_requests_v1)",
			"union":                "SELECT count() FROM gateway_requests_v1 UNION ALL SELECT count() FROM gateway_requests_per_hour_v1",
			"except":               "SELECT path FROM gateway_requests_v1 EXCEPT SELECT path FROM gateway_requests_v1",
			"scope filter":         "SELECT count() FROM gateway_requests_v1 WHERE project_id = 'proj_123' AND app_id = 'app_123' AND environment_id = 'env_123'",
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
