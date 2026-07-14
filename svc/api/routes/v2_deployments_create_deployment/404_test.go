package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestEnvironmentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	t.Run("unknown environment", func(t *testing.T) {
		capture.called = false
		req := imageRequest(t, setup.Project.Slug, setup.App.Slug, "does-not-exist", "nginx:latest")

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/environment_not_found", res.Body.Error.Type)
		require.False(t, capture.called)
	})

	t.Run("unknown project resolves to environment not found", func(t *testing.T) {
		capture.called = false
		req := imageRequest(t, "does-not-exist", setup.App.Slug, setup.Environment.Slug, "nginx:latest")

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/environment_not_found", res.Body.Error.Type)
		require.False(t, capture.called)
	})
}

func TestRedeployDeploymentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "d_does_not_exist")

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
	require.False(t, capture.called)
}

// TestRedeployCrossWorkspaceMasked verifies a deployment owned by another
// workspace is reported as not found, never as a 400, so the endpoint cannot
// confirm the existence of another tenant's deployment.
func TestRedeployCrossWorkspaceMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
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
		GitBranch:     "main",
	})

	attacker := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		ProjectSlug: "attacker-project",
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := deploymentRequest(t, attacker.Project.Slug, attacker.App.Slug, attacker.Environment.Slug, victimDep.ID)

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(attacker.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
	require.False(t, capture.called, "ctrl must not be called for a foreign deployment")
}

// TestRedeployWrongAppOrEnvironmentMasked verifies a deployment in the caller's
// own workspace but under a different app or environment is masked as not found,
// so the endpoint cannot probe for deployments across apps or projects the
// caller may not have access to.
func TestRedeployWrongAppOrEnvironmentMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	// A second app + environment + deployment in the SAME workspace.
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		Name:          "Other",
		Slug:          "other",
		DefaultBranch: "main",
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
		GitBranch:     "main",
	})

	// Redeploy that deployment while targeting the first app/environment.
	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, otherDep.ID)

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
	require.False(t, capture.called, "ctrl must not be called for a mismatched app or environment")
}
