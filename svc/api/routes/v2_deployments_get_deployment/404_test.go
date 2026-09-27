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

// TestDeploymentURNGrants covers the grants the dashboard proxy mints. A grant
// that does not cover this deployment gets the same masked 404 as a missing one.
func TestDeploymentURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	environment := urn.New().Workspace(setup.Workspace.ID).Project(setup.Project.ID).App(setup.App.ID).Environment(setup.Environment.ID)

	for _, tc := range []struct {
		name  string
		grant string
		found bool
	}{
		{name: "this deployment", grant: rbac.U(environment.Deployment(dep.ID), permissions.Read).Value, found: true},
		{name: "every deployment in the environment", grant: rbac.U(environment.Deployment("*"), permissions.Read).Value, found: true},
		{name: "every deployment in the workspace", grant: rbac.U(urn.New().Workspace(setup.Workspace.ID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Read).Value, found: true},
		{name: "another deployment", grant: rbac.U(environment.Deployment(uid.New(uid.DeploymentPrefix)), permissions.Read).Value, found: false},
		{name: "wrong action", grant: rbac.U(environment.Deployment(dep.ID), permissions.Delete).Value, found: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(setup.Workspace.ID, tc.grant)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{DeploymentId: dep.ID})
			if tc.found {
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, dep.ID)
		})
	}
}
