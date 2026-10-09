package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func TestUpdatePermissionAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	permission := seedPermission(h, workspace.ID)
	other := seedPermission(h, workspace.ID)
	permissionURN := func(permissionID, action string) string {
		return fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/%s#%s", workspace.ID, permission.ProjectID, permissionID, action)
	}

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "urn write on every project", permissions: []string{writeAnyPermission(workspace.ID)}, shouldPass: true},
		{name: "urn write on this permission", permissions: []string{permissionURN(permission.ID, "write")}, shouldPass: true},
		{name: "urn write on every permission", permissions: []string{permissionURN("*", "write")}, shouldPass: true},
		{name: "urn write alongside unrelated", permissions: []string{"api.*.read_api", permissionURN(permission.ID, "write")}, shouldPass: true},
		{name: "legacy update", permissions: []string{"rbac.*.update_permission"}, shouldPass: false},
		{name: "legacy read", permissions: []string{"rbac.*.read_permission"}, shouldPass: false},
		{name: "legacy create", permissions: []string{"rbac.*.create_permission"}, shouldPass: false},
		{name: "legacy update role", permissions: []string{"rbac.*.update_role"}, shouldPass: false},
		{name: "urn read on this permission", permissions: []string{permissionURN(permission.ID, "read")}, shouldPass: false},
		{name: "urn write on another permission", permissions: []string{permissionURN(other.ID, "write")}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(workspace.ID, tc.permissions...))
			name := uid.New("name")
			req := handler.Request{Permission: permission.ID, Name: &name}

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				require.Equal(t, name, findPermission(t, h, permission).Name)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, permission.ID)
			require.NotEqual(t, name, findPermission(t, h, permission).Name)
		})
	}
}

func TestUpdatePermissionExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	permission := seedPermission(h, workspace.ID)
	headers := authHeaders(h.CreateRootKey(workspace.ID, "rbac.*.read_permission"))
	name := "Read documents"

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Permission: permission.ID,
		Name:       &name,
	})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Permission: uid.New(uid.PermissionPrefix),
		Name:       &name,
	})

	require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotContains(t, realRes.RawBody, permission.ID)
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
}
