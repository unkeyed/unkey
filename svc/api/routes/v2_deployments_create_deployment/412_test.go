package handler_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestGitSourceWithoutRepoConnection(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	// No repo connection attached to the app.
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	req := gitRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, openapi.DeploymentSourceGit{
		Branch: ptr.P("main"),
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
}

// TestCreateDeploymentRequiresComputePlan and TestCreateDeploymentSpendSuspended
// cover the billing gate. This route is the only gate a caller sees: the create
// is submitted one-way, so the worker's own re-check lands long after the
// response. Both must reject before Restate is called at all.
func TestCreateDeploymentRequiresComputePlan(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	h.ClearComputePlanOverride(setup.Workspace.ID)

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "The workspace has no active Compute plan.", res.Body.Error.Detail)
}

// TestCreateDeploymentRejectsOversizedIdempotencyKey pins the bound: the key is
// hashed into the deployment id, so an unbounded header must be refused first.
func TestCreateDeploymentRejectsOversizedIdempotencyKey(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	headers := authHeaders(setup.RootKey)
	headers.Set("Idempotency-Key", strings.Repeat("k", 257))

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "Idempotency-Key")
}

func TestCreateDeploymentSpendSuspended(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	require.NoError(t, db.Query.UpsertWorkspaceBillingSpendSuspended(t.Context(), h.DB.RW(), db.UpsertWorkspaceBillingSpendSuspendedParams{
		WorkspaceID:    setup.Workspace.ID,
		SpendSuspended: true,
		CreatedAtM:     time.Now().UnixMilli(),
	}))

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, deploygate.StartSpendSuspended.Message(), res.Body.Error.Detail)
}
