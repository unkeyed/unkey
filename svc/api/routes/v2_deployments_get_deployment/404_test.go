package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_deployment"
)

func TestDeploymentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	req := handler.Request{DeploymentId: uid.New(uid.DeploymentPrefix)}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}

// A key without read_deployment must not learn whether the deployment exists:
// the handler masks the authorization failure as a 404.
func TestInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	req := handler.Request{DeploymentId: dep.ID}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}

// A deployment in another workspace must be indistinguishable from one that does
// not exist, so cross-workspace reads return 404 rather than leaking existence.
func TestDeploymentInAnotherWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	caller := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})
	other := h.CreateTestDeploymentSetup()

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   other.Workspace.ID,
		ProjectID:     other.Project.ID,
		AppID:         other.App.ID,
		EnvironmentID: other.Environment.ID,
	})

	req := handler.Request{DeploymentId: dep.ID}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(caller.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}

// TestGetDeploymentRejectsURNPermissionForAnotherResource verifies that a URN
// for another project, app, environment, workspace, or action returns not found.
func TestGetDeploymentRejectsURNPermissionForAnotherResource(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	permissionFor := func(workspaceID, projectID, appID, environmentID, deploymentID, action string) string {
		return fmt.Sprintf(
			"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s#%s",
			workspaceID,
			projectID,
			appID,
			environmentID,
			deploymentID,
			action,
		)
	}

	tests := []struct {
		name       string
		permission string
	}{
		{
			name: "wrong project",
			permission: permissionFor(setup.Workspace.ID, uid.New(uid.ProjectPrefix), setup.App.ID,
				setup.Environment.ID, dep.ID, "read"),
		},
		{
			name: "wrong app",
			permission: permissionFor(setup.Workspace.ID, setup.Project.ID, uid.New(uid.AppPrefix),
				setup.Environment.ID, dep.ID, "read"),
		},
		{
			name: "wrong environment",
			permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID,
				uid.New(uid.EnvironmentPrefix), dep.ID, "read"),
		},
		{
			name: "wrong workspace",
			permission: permissionFor(uid.New(uid.WorkspacePrefix), setup.Project.ID, setup.App.ID,
				setup.Environment.ID, dep.ID, "read"),
		},
		{
			name: "wrong action",
			permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID,
				setup.Environment.ID, dep.ID, "write"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(setup.Workspace.ID, test.permission)
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
				h,
				route,
				authHeaders(rootKey),
				handler.Request{DeploymentId: dep.ID},
			)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		})
	}
}
