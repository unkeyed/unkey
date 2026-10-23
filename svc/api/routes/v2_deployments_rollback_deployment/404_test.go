package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_rollback_deployment"
)

func TestRollbackDeploymentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.rollback_deployment"},
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: uid.New(uid.DeploymentPrefix),
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}

// A key without rollback_deployment must not learn whether the deployment
// exists: the handler masks the authorization failure as a 404.
func TestRollbackDeploymentInsufficientPermissionsMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
	})

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

// Legacy read_deployment is not accepted by the deployment read route, so it
// must not reveal whether this deployment exists.
func TestRollbackDeploymentLegacyReadDoesNotReveal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

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

func TestRollbackDeploymentReadVisibility(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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

	for _, tc := range []struct {
		name   string
		grant  string
		status int
	}{
		{"environment read", environmentRead(setup), http.StatusNotFound},
		{"deployment read", strings.TrimSuffix(environmentRead(setup), "#read") + "/deployments/" + dep.ID + "#read", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(setup.Workspace.ID, tc.grant)
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{DeploymentId: dep.ID})
			require.Equal(t, tc.status, res.Status, "%s", res.RawBody)
			if tc.status == http.StatusNotFound {
				missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{DeploymentId: uid.New(uid.DeploymentPrefix)})
				require.Equal(t, missing.Body.Error, res.Body.Error)
			} else {
				require.Contains(t, res.Body.Error.Detail, "rollback_deployment")
			}
		})
	}
}

// A deployment in another workspace must be indistinguishable from one that does
// not exist, so cross-workspace calls return 404 rather than leaking existence.
func TestRollbackDeploymentInAnotherWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	caller := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.rollback_deployment"},
	})
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
