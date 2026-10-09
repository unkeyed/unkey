package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_role"
)

func TestUpdateRoleAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	role := seedRole(h, workspace.ID)
	other := seedRole(h, workspace.ID)
	roleURN := func(roleID, action string) string {
		return fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/%s#%s", workspace.ID, role.ProjectID, roleID, action)
	}

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "urn write on every project", permissions: []string{writeAnyRole(workspace.ID)}, shouldPass: true},
		{name: "urn write on this role", permissions: []string{roleURN(role.ID, "write")}, shouldPass: true},
		{name: "urn write on every role", permissions: []string{roleURN("*", "write")}, shouldPass: true},
		{name: "urn write alongside unrelated", permissions: []string{"api.*.read_api", roleURN(role.ID, "write")}, shouldPass: true},
		{name: "legacy update role", permissions: []string{"rbac.*.update_role"}, shouldPass: false},
		{name: "legacy read role", permissions: []string{"rbac.*.read_role"}, shouldPass: false},
		{name: "legacy create role", permissions: []string{"rbac.*.create_role"}, shouldPass: false},
		{name: "legacy update permission", permissions: []string{"rbac.*.update_permission"}, shouldPass: false},
		{name: "urn read on this role", permissions: []string{roleURN(role.ID, "read")}, shouldPass: false},
		{name: "urn write on another role", permissions: []string{roleURN(other.ID, "write")}, shouldPass: false},
		{name: "urn write on every permission", permissions: []string{fmt.Sprintf("unkey:v1:%s:projects/*/rbac/permissions/*#write", workspace.ID)}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(workspace.ID, tc.permissions...))
			name := uid.New("name")
			req := handler.Request{Role: role.ID, Name: &name}

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				require.Equal(t, name, findRole(t, h, role).Name)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, role.ID)
			require.NotEqual(t, name, findRole(t, h, role).Name)
		})
	}
}

func TestUpdateRoleExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	role := seedRole(h, workspace.ID)
	headers := authHeaders(h.CreateRootKey(workspace.ID, "rbac.*.read_role"))
	name := uid.New("support.readonly")

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Role: role.ID,
		Name: &name,
	})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Role: uid.New(uid.RolePrefix),
		Name: &name,
	})

	require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotContains(t, realRes.RawBody, role.ID)
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
}
