package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test429_WorkspaceQueryQuota guarantees an exhausted workspace query quota
// maps to the public rate-limit response.
func Test429_WorkspaceQueryQuota(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueriesPerWindow(1))
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_gateway_requests")
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	req := Request{Query: "SELECT count() FROM gateway_requests_v1"}
	require.Equal(t, 200, testutil.CallRoute[Request, Response](h, route, auth(rootKey), req).Status)
	require.Equal(t, 429, testutil.CallRoute[Request, Response](h, route, auth(rootKey), req).Status)
}
