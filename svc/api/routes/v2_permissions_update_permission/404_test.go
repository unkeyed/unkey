package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func TestUpdatePermissionNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyPermission(workspace.ID)))
	name := "Read documents"

	t.Run("missing permission", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
			Permission: uid.New(uid.PermissionPrefix),
			Name:       &name,
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/permission_not_found", res.Body.Error.Type)
	})

	t.Run("permission in another workspace", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		permission := seedPermission(h, otherWorkspace.ID)

		for _, identifier := range []string{permission.ID, permission.Slug} {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				Permission: identifier,
				Name:       &name,
			})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		}
		require.Equal(t, permission.Name, findPermission(t, h, permission).Name)
	})
}
