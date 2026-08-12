package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// project.*.read_analytics is the permission of the gateway requests endpoint.
// It must not reach this data. Runtime logs contain the output of the customer's
// own code, thus read_logs is a separate action.
func Test403_UnrelatedPermissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)

	for _, permission := range []string{
		"project.*.read_analytics",
		"project.*.read_project",
		"project.*.read_deployment",
		"api.*.read_analytics",
		"ratelimit.*.read_analytics",
	} {
		t.Run(permission, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, permission)
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
				Query: "SELECT count() FROM runtime_logs_v1",
			})
			require.Equal(t, 403, res.Status)
		})
	}
}

// The handler refuses the caller before it opens a ClickHouse connection. Thus
// the status code cannot tell the caller whether the workspace has an analytics
// setup.
func Test403_ReturnsBeforeAnalyticsLookup(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_project")

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() FROM runtime_logs_v1",
	})
	require.Equal(t, 403, res.Status)
}
