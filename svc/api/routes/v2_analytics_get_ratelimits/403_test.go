package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test403_Permissions guarantees read_namespace and partially scoped analytics
// grants cannot authorize rate limit analytics queries.
func Test403_Permissions(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	first, second := createNamespace(t, h, workspaceID), createNamespace(t, h, workspaceID)
	tests := []struct{ key, query string }{
		{h.CreateRootKey(workspaceID, "ratelimit.*.read_namespace"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id = '%s'", first)},
		{h.CreateRootKey(workspaceID, "ratelimit."+first+".read_analytics"), fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id IN ('%s','%s')", first, second)},
	}
	for _, test := range tests {
		res := testutil.CallRoute[Request, Response](h, route, auth(test.key), Request{Query: test.query})
		require.Equal(t, 403, res.Status)
	}
}
