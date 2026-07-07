//go:build integration

package integration

import (
	"database/sql"
	"testing"
	"time"

	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployspendcheck"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployteardown"
)

// startSpendCheck wires the DeploySpendCheckService alongside the
// DeployTeardownService + DeploymentService it dispatches to, all under a real
// Restate server with a short drain poll/grace. The check reads no usage: the
// orchestrator prices workspaces and passes the gross in the request, so the
// tests drive the trip/resume decision through SpendMicroCents directly.
func startSpendCheck(t *testing.T, database db.Database) *restatetest.TestEnvironment {
	t.Helper()

	checkH, err := deployspendcheck.NewCheckHandler(deployspendcheck.CheckConfig{
		DB:             database,
		Admins:         workos.NewNoop(),
		Email:          email.NewNoop(),
		BillingBaseURL: "https://app.unkey.com",
	})
	require.NoError(t, err)

	teardownSvc, err := deployteardown.New(deployteardown.Config{
		DB:                database,
		DrainPollInterval: 200 * time.Millisecond,
		DrainGraceTimeout: 2 * time.Second,
	})
	require.NoError(t, err)

	return restatetest.Start(t,
		hydrav1.NewDeploymentServiceServer(deployment.New(deployment.Config{DB: database})),
		hydrav1.NewDeployTeardownServiceServer(teardownSvc),
		hydrav1.NewDeploySpendCheckServiceServer(checkH),
	)
}

// TestDeploySpendCheck_SuspendThenResume exercises the enforcement trigger
// end-to-end: an overage at/over budget with stop set suspends compute (the
// check dispatches Teardown(SUSPEND)), and a later run with a budget raised
// above the frozen overage resumes it (the check dispatches Resume).
func TestDeploySpendCheck_SuspendThenResume(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: db.DeploymentsDesiredStateRunning,
	}).Deployment

	// Make the deployment its app's current deployment so SUSPEND records it and
	// resume restores it.
	err := h.DB.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: dep.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
		AppID:               dep.AppID,
	})
	require.NoError(t, err)

	// A 200-cent gross (as the orchestrator would have priced it, in
	// micro-cents) is well over the tiny budget, so overage >= budget trips
	// the suspend.
	tEnv := startSpendCheck(t, h.DB)

	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	suspendResp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:             period,
		BudgetCents:        1,
		Stop:               true,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
		CurrentlySuspended: false,
	})
	require.NoError(t, err)
	require.True(t, suspendResp.GetSuspended(), "overage over budget with stop set should suspend")

	// Teardown clears current_deployment_id synchronously; the desired-state
	// change to stopped is applied asynchronously by the DeploymentService VO.
	require.Eventually(t, func() bool {
		app, getErr := h.DB.FindAppById(ctx, dep.AppID)
		if getErr != nil || app.CurrentDeploymentID.Valid {
			return false
		}
		got, getErr := h.DB.FindDeploymentById(ctx, dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateStopped
	}, 10*time.Second, 200*time.Millisecond, "compute should be suspended (current cleared, desired_state stopped)")

	// The check persisted the suspension to the workspace's billing row.
	billing, err := h.DB.FindWorkspaceBillingByWorkspaceID(ctx, dep.WorkspaceID)
	require.NoError(t, err)
	require.True(t, billing.SpendSuspended, "suspend should set spend_suspended")

	// A later run with a budget raised above the frozen overage resumes compute.
	// The orchestrator would now pass CurrentlySuspended=true from the column.
	resumeResp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:             period,
		BudgetCents:        1_000_000,
		Stop:               true,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
		CurrentlySuspended: true,
	})
	require.NoError(t, err)
	require.False(t, resumeResp.GetSuspended(), "overage under raised budget should resume")

	// Resume restores current_deployment_id and brings desired_state back to
	// running (the latter via the DeploymentService VO, asynchronously).
	require.Eventually(t, func() bool {
		app, getErr := h.DB.FindAppById(ctx, dep.AppID)
		if getErr != nil || !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String != dep.ID {
			return false
		}
		got, getErr := h.DB.FindDeploymentById(ctx, dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateRunning
	}, 10*time.Second, 200*time.Millisecond, "compute should be resumed (current restored, desired_state running)")

	// Resume cleared the billing row's flag.
	billing, err = h.DB.FindWorkspaceBillingByWorkspaceID(ctx, dep.WorkspaceID)
	require.NoError(t, err)
	require.False(t, billing.SpendSuspended, "resume should clear spend_suspended")
}

// TestDeploySpendCheck_ResumeOnBudgetRemoved exercises the budget-removal path:
// a suspended workspace whose budget was removed (BudgetCents=0) must still
// resume, since nothing caps its spend anymore. The orchestrator keeps it in
// the dispatch set via the deploy_spend_suspended column.
func TestDeploySpendCheck_ResumeOnBudgetRemoved(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: db.DeploymentsDesiredStateStopped,
	}).Deployment

	// Mark the workspace suspended, as a prior trip would have left it. The app
	// has no current deployment (suspend cleared it), so resume restores nothing;
	// the assertion is on the column clearing and the response.
	err := h.DB.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: true,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: h.Now()},
		ID:        dep.WorkspaceID,
	})
	require.NoError(t, err)

	tEnv := startSpendCheck(t, h.DB)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	// Budget removed (0) while suspended: the check resumes and clears the flag.
	resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:             period,
		BudgetCents:        0,
		Stop:               true,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
		CurrentlySuspended: true,
	})
	require.NoError(t, err)
	require.False(t, resp.GetSuspended(), "budget removed while suspended should resume")

	require.Eventually(t, func() bool {
		billing, getErr := h.DB.FindWorkspaceBillingByWorkspaceID(ctx, dep.WorkspaceID)
		return getErr == nil && !billing.SpendSuspended
	}, 10*time.Second, 200*time.Millisecond, "budget removal should clear spend_suspended")
}

// TestDeploySpendCheck_ResumeOnStopDisabled exercises turning off stopping
// while suspended and still over budget: compute must resume even though
// overage has not dropped below the budget.
func TestDeploySpendCheck_ResumeOnStopDisabled(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: db.DeploymentsDesiredStateStopped,
	}).Deployment

	err := h.DB.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: true,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: h.Now()},
		ID:        dep.WorkspaceID,
	})
	require.NoError(t, err)

	tEnv := startSpendCheck(t, h.DB)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:             period,
		BudgetCents:        1,
		Stop:               false,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
		CurrentlySuspended: true,
	})
	require.NoError(t, err)
	require.False(t, resp.GetSuspended(), "turning off stop while suspended should resume")

	require.Eventually(t, func() bool {
		billing, getErr := h.DB.FindWorkspaceBillingByWorkspaceID(ctx, dep.WorkspaceID)
		return getErr == nil && !billing.SpendSuspended
	}, 10*time.Second, 200*time.Millisecond, "stop disabled should clear spend_suspended")
}
