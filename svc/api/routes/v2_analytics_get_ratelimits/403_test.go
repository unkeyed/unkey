package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test403_Permissions guarantees read_namespace and partially scoped analytics
// grants cannot authorize rate limit analytics queries.
func Test403_Permissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	first, second := createNamespace(t, h, workspaceID), createNamespace(t, h, workspaceID)
	foreignWorkspace := h.CreateWorkspace()
	foreign := createNamespace(t, h, foreignWorkspace.ID)
	unknown := uid.New(uid.RatelimitNamespacePrefix)
	tests := []struct{ key, query string }{
		{h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id = '%s'", first)},
		{h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace"), "SELECT * FROM ratelimits_v1"},
		{h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id = '%s'", unknown)},
		{h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id = '%s'", foreign)},
		{h.CreateRootKey(workspaceID, "ratelimit."+first+".read_analytics"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id IN ('%s','%s')", first, second)},
		{h.CreateRootKey(workspaceID, "ratelimit."+first+".read_analytics"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id = '%s'", unknown)},
		{h.CreateRootKey(workspaceID, "ratelimit."+first+".read_analytics"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id = '%s'", foreign)},
	}
	for _, test := range tests {
		res := testutil.CallRoute[Request, Response](h, route, auth(test.key), Request{Query: test.query})
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
