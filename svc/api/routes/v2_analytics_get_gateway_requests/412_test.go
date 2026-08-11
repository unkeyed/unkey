package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test412_UnconfiguredAnalytics guarantees an authorized query fails when the
// workspace has no analytics setup.
func Test412_UnconfiguredAnalytics(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() FROM gateway_requests_v1",
	})
	require.Equal(t, 412, res.Status)
}
