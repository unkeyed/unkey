//go:build integration

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployspendcheck"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployteardown"
)

// fakeUsageReader returns a fixed active-keys count so the spend check prices a
// deterministic gross without a real ClickHouse. The harness provides MySQL
// only; the trip/resume decision is purely overage-vs-budget, so a stubbed
// usage row is enough to drive the full VO-to-VO suspend/resume path.
type fakeUsageReader struct {
	workspaceID string
	activeKeys  int64
}

func (f fakeUsageReader) GetInstanceMeterUsage(_ context.Context, _ clickhouse.GetInstanceMeterUsageRequest) ([]clickhouse.InstanceMeterUsage, error) {
	return nil, nil
}

func (f fakeUsageReader) GetActiveKeysUsage(_ context.Context, _ clickhouse.GetActiveKeysUsageRequest) ([]clickhouse.ActiveKeysUsage, error) {
	return []clickhouse.ActiveKeysUsage{
		{WorkspaceID: f.workspaceID, ActiveKeys: f.activeKeys},
	}, nil
}

// startSpendCheck wires the DeploySpendCheckService (with a fake usage reader)
// alongside the DeployTeardownService + DeploymentService it dispatches to, all
// under a real Restate server with a short drain poll/grace.
func startSpendCheck(t *testing.T, database db.Database, usage fakeUsageReader) *restatetest.TestEnvironment {
	t.Helper()

	checkH, err := deployspendcheck.NewCheckHandler(deployspendcheck.CheckConfig{
		DB:             database,
		Usage:          usage,
		Admins:         workos.New(""),
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
	err := db.Query.UpdateAppDeployments(ctx, h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: dep.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
		AppID:               dep.AppID,
	})
	require.NoError(t, err)

	// 1000 active keys price to 200 cents (0.2 cents/key), well over the tiny
	// budget, so overage >= budget trips the suspend.
	tEnv := startSpendCheck(t, h.DB, fakeUsageReader{workspaceID: dep.WorkspaceID, activeKeys: 1000})

	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	suspendResp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:              period,
		BudgetCents:         1,
		IncludedCreditCents: 0,
		Stop:                true,
		OrgId:               "org_test",
		WorkspaceName:       "test",
		WorkspaceSlug:       "test",
		CurrentlySuspended:  false,
	})
	require.NoError(t, err)
	require.True(t, suspendResp.GetSuspended(), "overage over budget with stop set should suspend")

	// Teardown clears current_deployment_id synchronously; the desired-state
	// change to stopped is applied asynchronously by the DeploymentService VO.
	require.Eventually(t, func() bool {
		app, getErr := db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
		if getErr != nil || app.CurrentDeploymentID.Valid {
			return false
		}
		got, getErr := db.Query.FindDeploymentById(ctx, h.DB.RO(), dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateStopped
	}, 10*time.Second, 200*time.Millisecond, "compute should be suspended (current cleared, desired_state stopped)")

	// The check persisted the suspension to the workspace's column.
	ws, err := db.Query.FindWorkspaceByID(ctx, h.DB.RO(), dep.WorkspaceID)
	require.NoError(t, err)
	require.True(t, ws.DeploySpendSuspended, "suspend should set deploy_spend_suspended")

	// A later run with a budget raised above the frozen overage resumes compute.
	// The orchestrator would now pass CurrentlySuspended=true from the column.
	resumeResp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:              period,
		BudgetCents:         1_000_000,
		IncludedCreditCents: 0,
		Stop:                true,
		OrgId:               "org_test",
		WorkspaceName:       "test",
		WorkspaceSlug:       "test",
		CurrentlySuspended:  true,
	})
	require.NoError(t, err)
	require.False(t, resumeResp.GetSuspended(), "overage under raised budget should resume")

	// Resume restores current_deployment_id and brings desired_state back to
	// running (the latter via the DeploymentService VO, asynchronously).
	require.Eventually(t, func() bool {
		app, getErr := db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
		if getErr != nil || !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String != dep.ID {
			return false
		}
		got, getErr := db.Query.FindDeploymentById(ctx, h.DB.RO(), dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateRunning
	}, 10*time.Second, 200*time.Millisecond, "compute should be resumed (current restored, desired_state running)")

	// Resume cleared the column.
	ws, err = db.Query.FindWorkspaceByID(ctx, h.DB.RO(), dep.WorkspaceID)
	require.NoError(t, err)
	require.False(t, ws.DeploySpendSuspended, "resume should clear deploy_spend_suspended")
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
	err := db.Query.SetWorkspaceDeploySpendSuspended(ctx, h.DB.RW(), db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: true,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: h.Now()},
		ID:        dep.WorkspaceID,
	})
	require.NoError(t, err)

	tEnv := startSpendCheck(t, h.DB, fakeUsageReader{workspaceID: dep.WorkspaceID, activeKeys: 1000})
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	// Budget removed (0) while suspended: the check resumes and clears the flag.
	resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:              period,
		BudgetCents:         0,
		IncludedCreditCents: 0,
		Stop:                true,
		OrgId:               "org_test",
		WorkspaceName:       "test",
		WorkspaceSlug:       "test",
		CurrentlySuspended:  true,
	})
	require.NoError(t, err)
	require.False(t, resp.GetSuspended(), "budget removed while suspended should resume")

	require.Eventually(t, func() bool {
		ws, getErr := db.Query.FindWorkspaceByID(ctx, h.DB.RO(), dep.WorkspaceID)
		return getErr == nil && !ws.DeploySpendSuspended
	}, 10*time.Second, 200*time.Millisecond, "budget removal should clear deploy_spend_suspended")
}
