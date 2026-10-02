package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test403_URNNamespaceReadDoesNotAllowLogRead guarantees parent resource
// access does not imply access to rate limit analytics.
func Test403_URNNamespaceReadDoesNotAllowLogRead(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	projectID := createProject(t, h, workspaceID)
	namespaceID := createNamespaceInProject(t, h, workspaceID, projectID, uid.New("test"))
	namespace := urn.New().Workspace(workspaceID).Project(projectID).RatelimitNamespace(namespaceID)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("%s#%s", namespace.String(), permissions.Read))

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT namespace_id FROM ratelimits_v1"})
	require.Equal(t, 403, res.Status)
}

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
