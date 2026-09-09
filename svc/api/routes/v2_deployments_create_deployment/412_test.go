package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestGitSourceWithoutRepoConnection(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.RejectingDeployRestate(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_REPO_CONNECTION, ""))
	h.Register(route)

	// No repo connection attached to the app.
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := gitRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, openapi.DeploymentSourceGit{
		Branch: ptr.P("main"),
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
}

// The billing gate lives in the create worker and answers a refusal as an
// outcome. These pin the 412 and message the route turns each outcome into.
func TestNoComputePlanOutcomeIs412(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.RejectingDeployRestate(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NO_COMPUTE_PLAN, ""))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "The workspace has no active Compute plan.", res.Body.Error.Detail)
}

func TestSpendSuspendedOutcomeIs412(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.RejectingDeployRestate(t, hydrav1.CreateOutcome_CREATE_OUTCOME_SPEND_SUSPENDED, ""))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, deploygate.StartSpendSuspended.Message(), res.Body.Error.Detail)
}

func TestCommitNotResolvedOutcomeIs412(t *testing.T) {
	const detail = `branch "release" or commit "" not found in acme/api`
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.RejectingDeployRestate(t, hydrav1.CreateOutcome_CREATE_OUTCOME_COMMIT_NOT_RESOLVED, detail))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, detail)
}

func TestNewerDeploymentOutcomeIs412(t *testing.T) {
	const detail = `a newer active deployment exists on branch "main"`
	h := testutil.NewHarness(t)
	route := newRoute(h, testutil.RejectingDeployRestate(t, hydrav1.CreateOutcome_CREATE_OUTCOME_NEWER_DEPLOYMENT_EXISTS, detail))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, detail)
}
