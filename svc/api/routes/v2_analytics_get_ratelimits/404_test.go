package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Test404_UnknownAndForeignNamespacesAreIndistinguishable guarantees namespace
// resolution does not reveal whether an ID belongs to another workspace.
func Test404_UnknownAndForeignNamespacesAreIndistinguishable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	owned := createNamespace(t, h, workspaceID)
	foreignWorkspace := h.CreateWorkspace()
	foreign := createNamespace(t, h, foreignWorkspace.ID)
	rootKey := h.CreateRootKey(workspaceID, "ratelimit.*.read_analytics")
	var details []string
	for _, id := range []string{uid.New(uid.RatelimitNamespacePrefix), foreign} {
		res := testutil.CallRoute[Request, openapi.NotFoundErrorResponse](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT * FROM ratelimits_v1 WHERE namespace_id IN ('%s','%s')", owned, id)})
		require.Equal(t, 404, res.Status)
		details = append(details, res.Body.Error.Type)
	}
	require.Equal(t, details[0], details[1])
}
