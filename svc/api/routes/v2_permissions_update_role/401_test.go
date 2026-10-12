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

func TestUpdateRoleUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	role := seedRole(h, h.Resources().UserWorkspace.ID)
	name := uid.New("support.readonly")

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, authHeaders("invalid_token"), handler.Request{
		Role: role.ID,
		Name: &name,
	})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
	require.Equal(t, role.Name, findRole(t, h, role).Name)
}
