package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Test400_QueryBeyondRetention guarantees a lower time bound older than the
// workspace retention window is refused before it reaches ClickHouse.
func Test400_QueryBeyondRetention(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	// The default workspace retention is 30 days.
	for name, query := range map[string]string{
		"raw table":     "SELECT count() AS total FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 60 DAY)",
		"minute rollup": "SELECT count() AS total FROM gateway_requests_per_minute_v1 WHERE time >= now() - INTERVAL 60 DAY",
		"hour rollup":   "SELECT count() AS total FROM gateway_requests_per_hour_v1 WHERE time >= now() - INTERVAL 60 DAY",
		"day rollup":    "SELECT count() AS total FROM gateway_requests_per_day_v1 WHERE time >= now() - INTERVAL 60 DAY",
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 400, res.Status)
			require.Contains(t, res.Body.Error.Type, "query_range_exceeds_retention")
			require.Contains(t, res.Body.Error.Detail, "30 days")
		})
	}
}

// Test400_QueryBeyondCustomRetention guarantees a longer workspace retention
// setting moves the boundary rather than removing it.
func Test400_QueryBeyondCustomRetention(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90))
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_analytics")
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, auth(rootKey), Request{
		Query: "SELECT count() AS total FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 120 DAY)",
	})
	require.Equal(t, 400, res.Status)
	require.Contains(t, res.Body.Error.Type, "query_range_exceeds_retention")
	require.Contains(t, res.Body.Error.Detail, "90 days")
}
