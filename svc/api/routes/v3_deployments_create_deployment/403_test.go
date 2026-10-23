package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

func TestMissingPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	for _, environment := range []string{setup.Environment.Slug, "does-not-exist"} {
		t.Run(environment, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
				Project:     setup.Project.Slug,
				App:         setup.App.Slug,
				Environment: environment,
				Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
			})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "The requested project, app, or environment does not exist.", res.Body.Error.Detail)
			require.NotContains(t, res.RawBody, setup.Environment.ID)
		})
	}
}

func TestNoAccessIsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "The requested project, app, or environment does not exist.", res.Body.Error.Detail)
}

func TestEnvironmentReaderIsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	environment := urn.New().Workspace(setup.Workspace.ID).Project(setup.Project.ID).App(setup.App.ID).Environment(setup.Environment.ID)
	rootKey := h.CreateRootKey(setup.Workspace.ID, environment.String()+"#read")

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "#write")
	require.Contains(t, res.Body.Error.Detail, "Log in again")
	require.NotContains(t, res.RawBody, setup.Environment.ID)
}
