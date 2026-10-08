package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func TestUpdatePermissionSlugConflict(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyPermission(workspace.ID)))
	permission := seedPermission(h, workspace.ID)
	taken := seedPermission(h, workspace.ID)
	name := "Read documents"

	res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{
		Permission: permission.ID,
		Name:       &name,
		Slug:       &taken.Slug,
	})
	require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, taken.Slug)

	stored := findPermission(t, h, permission)
	require.Equal(t, permission.Slug, stored.Slug)
	require.Equal(t, permission.Name, stored.Name)
}
