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
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

func TestStartDeploymentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, writeDeployments(setup.Workspace.ID))

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: uid.New(uid.DeploymentPrefix),
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}

// A key that can neither write nor read the deployment must not learn whether
// it exists: the handler masks the authorization failure as a 404.
func TestStartDeploymentInsufficientPermissionsMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, rbac.U(urn.New().Workspace(setup.Workspace.ID).Project("*").App("*").Environment("*"), permissions.Read).Value)

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, dep.ID)
}

func TestStartDeploymentReaderIsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentRead(setup))

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, authHeaders(rootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "#write")
	require.Contains(t, res.Body.Error.Detail, "Log in again")
}

// A deployment in another workspace must be indistinguishable from one that does
// not exist, so cross-workspace calls return 404 rather than leaking existence.
func TestStartDeploymentInAnotherWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	caller := h.CreateTestDeploymentSetup()
	caller.RootKey = h.CreateRootKey(caller.Workspace.ID, writeDeployments(caller.Workspace.ID))
	other := h.CreateTestDeploymentSetup()

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   other.Workspace.ID,
		ProjectID:     other.Project.ID,
		AppID:         other.App.ID,
		EnvironmentID: other.Environment.ID,
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(caller.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}
