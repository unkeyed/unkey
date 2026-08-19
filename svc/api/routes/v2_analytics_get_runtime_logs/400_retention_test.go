package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func Test400_QueryBeyondRetention(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_runtime_logs")

	// The default workspace retention is 30 days.
	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, auth(rootKey), Request{
		Query: "SELECT count() AS total FROM runtime_logs_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 60 DAY)",
	})
	require.Equal(t, 400, res.Status)
	require.Contains(t, res.Body.Error.Type, "query_range_exceeds_retention")
	require.Contains(t, res.Body.Error.Detail, "30 days")
}

func Test400_QueryBeyondCustomRetention(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90))
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_runtime_logs")
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, auth(rootKey), Request{
		Query: "SELECT count() AS total FROM runtime_logs_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 120 DAY)",
	})
	require.Equal(t, 400, res.Status)
	require.Contains(t, res.Body.Error.Type, "query_range_exceeds_retention")
	require.Contains(t, res.Body.Error.Detail, "90 days")
}
