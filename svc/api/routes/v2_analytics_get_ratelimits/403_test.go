package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test403_Permissions guarantees unrelated permissions cannot authorize rate
// limit analytics queries.
func Test403_Permissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace")
	for _, query := range []string{
		"SELECT * FROM ratelimits_v1",
		"SELECT * FROM ratelimits_v1 WHERE namespace_id = 'rlns_unknown'",
	} {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
		require.Equal(t, 403, res.Status)
	}
}

// Test403_NoPermissionsReturnsBeforeAnalyticsLookup guarantees principals
// without analytics permissions are rejected before resolving ClickHouse.
func Test403_NoPermissionsReturnsBeforeAnalyticsLookup(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace")

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT * FROM ratelimits_v1"})
	require.Equal(t, 403, res.Status)
}
