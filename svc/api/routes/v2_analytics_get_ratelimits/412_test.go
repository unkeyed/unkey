package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test412_UnconfiguredAnalytics guarantees an authorized query fails when
// analytics is not configured for the workspace.
func Test412_UnconfiguredAnalytics(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT * FROM ratelimits_v1 WHERE namespace_id = 'rlns_missing'"})
	require.Equal(t, 412, res.Status)
}
