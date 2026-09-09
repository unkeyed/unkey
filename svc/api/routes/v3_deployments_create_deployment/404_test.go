package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

func TestEnvironmentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	t.Run("unknown environment", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
			Project:     setup.Project.Slug,
			App:         setup.App.Slug,
			Environment: "does-not-exist",
			Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/environment_not_found", res.Body.Error.Type)
	})

	t.Run("unknown project resolves to environment not found", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
			Project:     "does-not-exist",
			App:         setup.App.Slug,
			Environment: setup.Environment.Slug,
			Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/environment_not_found", res.Body.Error.Type)
	})
}

func TestRedeployDeploymentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: "d_does_not_exist"},
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
}

// TestRedeployCrossWorkspaceMasked verifies a deployment owned by another
// workspace is reported as not found, never as a 400, so the endpoint cannot
// confirm the existence of another tenant's deployment.
func TestRedeployCrossWorkspaceMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	victim := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		ProjectSlug: "victim-project",
	})
	victimDep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   victim.Workspace.ID,
		ProjectID:     victim.Project.ID,
		AppID:         victim.App.ID,
		EnvironmentID: victim.Environment.ID,
	})

	attacker := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		ProjectSlug: "attacker-project",
		Permissions: []string{"environment.*.create_deployment"},
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(attacker.RootKey), handler.Request{
		Project:     attacker.Project.Slug,
		App:         attacker.App.Slug,
		Environment: attacker.Environment.Slug,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: victimDep.ID},
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
}

// TestRedeployWrongAppOrEnvironmentMasked verifies a deployment in the caller's
// own workspace but under a different app or environment is masked as not found,
// so the endpoint cannot probe for deployments across apps or projects the
// caller may not have access to.
func TestRedeployWrongAppOrEnvironmentMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		Name:        "Other",
		Slug:        "other",
	})
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       otherApp.ID,
		Slug:        "staging",
		Description: "staging environment",
	})
	otherDep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         otherApp.ID,
		EnvironmentID: otherEnv.ID,
	})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: otherDep.ID},
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
}
