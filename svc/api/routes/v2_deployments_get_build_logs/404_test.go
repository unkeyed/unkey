package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_build_logs"
)

func TestGetBuildLogsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})
	dep := createDeployment(h, setup)
	otherDep := createDeployment(h, h.CreateTestDeploymentSetup())

	call := func(t *testing.T, rootKey, deploymentID string) testutil.TestResponse[openapi.NotFoundErrorResponse] {
		t.Helper()
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{DeploymentId: deploymentID})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "The requested deployment does not exist.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, deploymentID)
		return res
	}

	t.Run("unknown deployment", func(t *testing.T) {
		call(t, setup.RootKey, uid.New(uid.DeploymentPrefix))
	})

	t.Run("deployment in another workspace", func(t *testing.T) {
		call(t, setup.RootKey, otherDep.deploymentID)
	})

	permissionFor := func(workspaceID, projectID, appID, environmentID, deploymentID string, action permissions.Action) string {
		return rbac.U(urn.New().Workspace(workspaceID).Project(projectID).App(appID).Environment(environmentID).Deployment(deploymentID), action).Value
	}
	for _, tc := range []struct {
		name       string
		permission string
	}{
		{name: "other action on deployments", permission: "environment.*.create_deployment"},
		{name: "other environment", permission: fmt.Sprintf("environment.%s.read_deployment", uid.New(uid.EnvironmentPrefix))},
		{name: "urn for another deployment", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID, uid.New(uid.DeploymentPrefix), permissions.Read)},
		{name: "urn for another project", permission: permissionFor(setup.Workspace.ID, uid.New(uid.ProjectPrefix), setup.App.ID, setup.Environment.ID, dep.deploymentID, permissions.Read)},
		{name: "urn for another app", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, uid.New(uid.AppPrefix), setup.Environment.ID, dep.deploymentID, permissions.Read)},
		{name: "urn for another environment", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID, uid.New(uid.EnvironmentPrefix), dep.deploymentID, permissions.Read)},
		{name: "urn for another workspace", permission: permissionFor(uid.New(uid.WorkspacePrefix), setup.Project.ID, setup.App.ID, setup.Environment.ID, dep.deploymentID, permissions.Read)},
		{name: "urn with wrong action", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID, dep.deploymentID, permissions.Delete)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call(t, h.CreateRootKey(setup.Workspace.ID, tc.permission), dep.deploymentID)
		})
	}

	t.Run("existence is not leaked to a key without grants", func(t *testing.T) {
		rootKey := h.CreateRootKey(setup.Workspace.ID)
		existing := call(t, rootKey, dep.deploymentID)
		missing := call(t, rootKey, uid.New(uid.DeploymentPrefix))
		require.Equal(t, missing.Body.Error.Detail, existing.Body.Error.Detail)
		require.Equal(t, missing.Body.Error.Type, existing.Body.Error.Type)
		require.Equal(t, missing.Body.Error.Status, existing.Body.Error.Status)
	})
}
