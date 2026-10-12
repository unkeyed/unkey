package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_role"
)

func TestUpdateRoleNameConflict(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyRole(workspace.ID)))
	role := seedRole(h, workspace.ID)
	taken := seedRole(h, workspace.ID)

	for _, name := range []string{taken.Name, strings.ToUpper(taken.Name)} {
		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{
			Role: role.ID,
			Name: &name,
		})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Detail, name)
	}

	stored := findRole(t, h, role)
	require.Equal(t, role.Name, stored.Name)
	require.Equal(t, role.Description.String, stored.Description.String)
}
