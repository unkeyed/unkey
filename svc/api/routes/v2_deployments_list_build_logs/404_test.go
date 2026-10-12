package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_build_logs"
)

func TestListBuildLogsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := buildLogsRootKey(h, setup)
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
		call(t, rootKey, uid.New(uid.DeploymentPrefix))
	})

	t.Run("deployment in another workspace", func(t *testing.T) {
		call(t, rootKey, otherDep.deploymentID)
	})

	permissionFor := func(workspaceID, projectID, appID, environmentID, deploymentID string, action permissions.Action) string {
		return rbac.U(urn.New().Workspace(workspaceID).Project(projectID).App(appID).Environment(environmentID).Deployment(deploymentID).BuildLogs(), action).Value
	}
	deployment := urn.New().Workspace(setup.Workspace.ID).Project(setup.Project.ID).App(setup.App.ID).Environment(setup.Environment.ID).Deployment(dep.deploymentID)
	for _, tc := range []struct {
		name       string
		permission string
	}{
		{name: "legacy deployment read", permission: "environment.*.read_deployment"},
		{name: "deployment read", permission: rbac.U(deployment, permissions.Read).Value},
		{name: "runtime log read", permission: rbac.U(deployment.Logs(), permissions.Read).Value},
		{name: "urn for another deployment", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID, uid.New(uid.DeploymentPrefix), permissions.Read)},
		{name: "urn for another project", permission: permissionFor(setup.Workspace.ID, uid.New(uid.ProjectPrefix), setup.App.ID, setup.Environment.ID, dep.deploymentID, permissions.Read)},
		{name: "urn for another app", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, uid.New(uid.AppPrefix), setup.Environment.ID, dep.deploymentID, permissions.Read)},
		{name: "urn for another environment", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID, uid.New(uid.EnvironmentPrefix), dep.deploymentID, permissions.Read)},
		{name: "urn for another workspace", permission: permissionFor(uid.New(uid.WorkspacePrefix), setup.Project.ID, setup.App.ID, setup.Environment.ID, dep.deploymentID, permissions.Read)},
		{name: "urn with wrong action", permission: permissionFor(setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID, dep.deploymentID, permissions.Write)},
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
