package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// No permission other than the runtime logs wildcard can read this data. The
// gateway request wildcard must not carry over, because it scopes a different
// dataset.
func Test403_UnrelatedPermissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)

	for _, permission := range []string{
		fmt.Sprintf("unkey:v1:%s:projects/*/keyspaces/*/logs#read", workspaceID),
		fmt.Sprintf("unkey:v1:%s:projects/*/ratelimits/namespaces/*/logs#read", workspaceID),
		fmt.Sprintf("unkey:v1:%s:projects/*#read", workspaceID),
		fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/deployments/*#read", workspaceID),
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
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:projects/*#read", workspaceID))

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() FROM runtime_logs_v1",
	})
	require.Equal(t, 403, res.Status)
}
