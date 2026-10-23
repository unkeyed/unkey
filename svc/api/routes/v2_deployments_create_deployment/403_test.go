package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.UncalledDeployRestate(t))
	h.Register(route)

	// A key with an unrelated permission on the project.
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, setup.Environment.ID)
	require.Contains(t, res.RawBody, "does not exist")
}

func TestEnvironmentReaderIsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.UncalledDeployRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	environment := urn.New().Workspace(setup.Workspace.ID).Project(setup.Project.ID).App(setup.App.ID).Environment(setup.Environment.ID)
	rootKey := h.CreateRootKey(setup.Workspace.ID, environment.String()+"#read")

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "#write")
	require.Contains(t, res.Body.Error.Detail, "Log in again")
	require.NotContains(t, res.RawBody, setup.Environment.ID)
}
