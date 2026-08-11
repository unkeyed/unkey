package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test403_UnrelatedPermissions guarantees no permission other than the gateway
// analytics wildcard can read this data. The analytics wildcards of the sibling
// endpoints must not carry over, because they scope different datasets.
func Test403_UnrelatedPermissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)

	for _, permission := range []string{
		"project.*.read_project",
		"project.*.read_deployment",
		"api.*.read_analytics",
		"ratelimit.*.read_analytics",
	} {
		t.Run(permission, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, permission)
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
				Query: "SELECT count() FROM gateway_requests_v1",
			})
			require.Equal(t, 403, res.Status)
		})
	}
}

// Test403_ReturnsBeforeAnalyticsLookup guarantees an unauthorized caller is
// refused before the handler opens a ClickHouse connection, so a missing
// analytics setup cannot mask a permission failure.
func Test403_ReturnsBeforeAnalyticsLookup(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_project")

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() FROM gateway_requests_v1",
	})
	require.Equal(t, 403, res.Status)
}
