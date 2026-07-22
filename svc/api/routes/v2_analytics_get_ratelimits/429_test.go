package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test429_WorkspaceQueryQuota guarantees exhausting the workspace query quota
// maps to the public rate-limit response.
func Test429_WorkspaceQueryQuota(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueriesPerWindow(1))
	id := createNamespace(t, h, workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "ratelimit.*.read_analytics")
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)
	req := Request{Query: fmt.Sprintf("SELECT count(*) FROM ratelimits_v1 WHERE namespace_id = '%s'", id)}
	require.Equal(t, 200, testutil.CallRoute[Request, Response](h, route, auth(rootKey), req).Status)
	require.Equal(t, 429, testutil.CallRoute[Request, Response](h, route, auth(rootKey), req).Status)
}
