package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test412_UnconfiguredPrecedesNamespaceAndPermissionChecks(t *testing.T) {
	h, route, workspaceID := newRoute(t, false)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace")
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: "SELECT * FROM ratelimits_v1 WHERE namespace_id = 'rlns_missing'"})
	require.Equal(t, 412, res.Status)
}
