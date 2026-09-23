package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test403_UnrelatedPermissions guarantees no permission other than the gateway
// request wildcard can read this data. The analytics wildcards of the other
// resources must not carry over, because they scope different datasets.
func Test403_UnrelatedPermissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)

	for _, permission := range []string{
		"project.*.read_project",
		"project.*.read_deployment",
		"project.*.read_analytics",
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
	for name, permission := range map[string]func(string) string{
		"legacy unrelated action": func(string) string { return "project.*.read_project" },
		"canonical wrong action": func(workspaceID string) string {
			return fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/gateway/logs#write", workspaceID)
		},
		"canonical wrong workspace": func(string) string {
			return "unkey:v1:ws_foreign:projects/*/apps/*/environments/*/gateway/logs#read"
		},
	} {
		t.Run(name, func(t *testing.T) {
			h, route, workspaceID := newRoute(t, false)
			rootKey := h.CreateRootKey(workspaceID, permission(workspaceID))

			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
				Query: "SELECT count() FROM gateway_requests_v1",
			})
			require.Equal(t, 403, res.Status)
		})
	}
}
