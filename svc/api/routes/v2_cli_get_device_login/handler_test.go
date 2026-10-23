package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_get_device_login"
)

// TestGrantablePermissionsFollowsEachRole guarantees the approval page offers
// exactly the permissions of the signed-in role. Admin can grant workspace
// root-key access. Developer and basic_member share one set and cannot grant
// root keys. Viewer is read-only.
func TestGrantablePermissionsFollowsEachRole(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	admin := workos.PermissionsForRoles(workspaceID, []string{"admin"})
	developer := workos.PermissionsForRoles(workspaceID, []string{"developer"})
	member := workos.PermissionsForRoles(workspaceID, []string{"basic_member"})
	viewer := workos.PermissionsForRoles(workspaceID, []string{"viewer"})

	require.Equal(t, admin, handler.GrantablePermissions(principalWith(admin)))
	require.Equal(t, developer, handler.GrantablePermissions(principalWith(developer)))
	require.Equal(t, member, handler.GrantablePermissions(principalWith(member)))
	require.Equal(t, viewer, handler.GrantablePermissions(principalWith(viewer)))
	require.Empty(t, handler.GrantablePermissions(principalWith(nil)))

	require.Equal(t, developer, member)
	require.Contains(t, admin, "unkey:v1:"+workspaceID+":**#*")
	require.NotContains(t, developer, "unkey:v1:"+workspaceID+":**#*")
	for _, permission := range developer {
		require.NotContains(t, permission, "rootKeys")
	}
	for _, permission := range viewer {
		require.False(t, strings.HasSuffix(permission, "#write"))
		require.False(t, strings.HasSuffix(permission, "#delete"))
		require.False(t, strings.HasSuffix(permission, "#decrypt"))
	}
}

func principalWith(permissions []string) *principal.Principal {
	return &principal.Principal{Permissions: permissions}
}

func TestGetRequiresADashboardSession(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.52", UserAgent: ""})
	require.NoError(t, err)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", "unkey:v1:"+workspace.ID+":rootKeys/*#read")
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + rootKey}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: started.UserCode})
	require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
}

func TestGetRejectsAnInvalidBearer(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer invalid"}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: "TEST-0001"})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
}
