package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestGitSourceWithoutRepoConnection(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newRejectingRestate(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION))
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
// cover the billing gate. The gate itself belongs to the create worker, which
// answers a refusal as an enum; this route awaits that answer and is what turns
// it into a 412 a caller can act on.
func TestCreateDeploymentRequiresComputePlan(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newRejectingRestate(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_COMPUTE_PLAN))
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

func TestCreateDeploymentSpendSuspended(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newRejectingRestate(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SPEND_SUSPENDED))
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
