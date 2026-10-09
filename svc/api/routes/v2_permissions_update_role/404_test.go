package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_role"
)

func TestUpdateRoleNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyRole(workspace.ID)))
	name := uid.New("support.readonly")

	t.Run("missing role", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
			Role: uid.New(uid.RolePrefix),
			Name: &name,
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/role_not_found", res.Body.Error.Type)
	})

	t.Run("role in another workspace", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		role := seedRole(h, otherWorkspace.ID)

		for _, identifier := range []string{role.ID, role.Name} {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				Role: identifier,
				Name: &name,
			})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		}
		require.Equal(t, role.Name, findRole(t, h, role).Name)
	})
}
