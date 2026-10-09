package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func TestUpdatePermissionUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	permission := seedPermission(h, h.Resources().UserWorkspace.ID)
	name := "Read documents"

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, authHeaders("invalid_token"), handler.Request{
		Permission: permission.ID,
		Name:       &name,
	})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
	require.Equal(t, permission.Name, findPermission(t, h, permission).Name)
}
