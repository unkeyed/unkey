//go:build integration

package integration

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/email"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// setAppCurrent points an app's current deployment at dep, the precondition a
// SUSPEND teardown needs to record (and Resume to restore) that deployment.
func setAppCurrent(t *testing.T, h *Harness, appID, deploymentID string) {
	t.Helper()
	require.NoError(t, h.DB.UpdateAppDeployments(h.Context(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
		AppID:               appID,
	}))
}

// TestDeploySpendCheck_ReEnforcesLeakedCompute covers re-enforcement: once
// deploy_spend_suspended is true the pre-fix check never re-sent Teardown, so a
// deployment left running (a kill between the flag write and the first teardown,
// a drain-grace timeout, or a create racing the gate) kept accruing spend while
// the dashboard said stopped. A tick that finds the workspace suspended, still
// over budget, and stop still enabled must re-enforce by re-sending Teardown,
// and must not re-send the stopped email (it is not a new suspension).
func TestDeploySpendCheck_ReEnforcesLeakedCompute(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})
	setAppCurrent(t, h, dep.AppID, dep.ID)

	// Torn state: the column says suspended but compute is still running.
	require.NoError(t, h.DB.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: true,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: h.Now()},
		ID:        dep.WorkspaceID,
	}))

	sender := email.NewCapture()
	tEnv := startSpendCheckCapturing(t, h.DB, sender)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:             period,
		BudgetCents:        1,
		Stop:               true,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
		CurrentlySuspended: true,
	})
	require.NoError(t, err)
	require.True(t, resp.GetSuspended(), "an over-budget suspended workspace stays suspended")

	// Re-enforcement re-sent Teardown, which stops the leaked deployment
	// (current cleared, desired_state stopped), exactly as the first suspend would.
	require.Eventually(t, func() bool {
		app, e := h.DB.FindAppById(ctx, dep.AppID)
		if e != nil || app.CurrentDeploymentID.Valid {
			return false
		}
		got, e := h.DB.FindDeploymentById(ctx, dep.ID)
		return e == nil && got.DesiredState == mysqltype.DeploymentsDesiredStateStopped
	}, 15*time.Second, 200*time.Millisecond, "re-enforcement should stop leaked compute")

	// Re-enforcement is not a new suspension transition, so no email fires.
	require.Equal(t, 0, sender.CountByTemplate(templateBudgetStopped),
		"re-enforcement must not resend the stopped email")
	require.Equal(t, 0, sender.CountByTemplate(templateBudgetAlert),
		"a suspended workspace gets no threshold warning")
}

// TestDeploySpendCheck_SkipsSuspendAfterCancel covers the cancel race: a spend
// check dispatched from a pre-cancel snapshot (stop set, over budget, not yet
// suspended) can arrive after cancelDeploy already cleared the plan and the
// spend-suspended flag. Suspending then would re-set the flag on a plan-less
// workspace and strand it so a later resubscribe starts blocked. The check
// re-reads the live entitlement and skips enforcement when the plan is gone.
func TestDeploySpendCheck_SkipsSuspendAfterCancel(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})
	setAppCurrent(t, h, dep.AppID, dep.ID)

	// Simulate the cancel that raced ahead of this check: deprovisionCompute
	// clears the plan, so the workspace is no longer Deploy-entitled.
	require.NoError(t, h.DB.ClearWorkspaceDeployPlan(ctx, db.ClearWorkspaceDeployPlanParams{
		ID:        dep.WorkspaceID,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: h.Now()},
	}))

	tEnv := startSpendCheck(t, h.DB)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	// The stale snapshot's willSuspend condition: over budget, stop set, not yet
	// suspended. Pre-fix this suspended a plan-less workspace.
	resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
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
	require.False(t, resp.GetSuspended(), "a cancelled (plan-less) workspace must not be spend-suspended")

	// The flag stays cleared in the database, so a later resubscribe is not blocked.
	billing, err := h.DB.FindWorkspaceBillingByWorkspaceID(ctx, dep.WorkspaceID)
	require.NoError(t, err)
	require.False(t, billing.SpendSuspended, "spend-suspended must stay cleared for a plan-less workspace")
}

// TestDeploySpendCheck_ReEnforceMergesSuspensionRecord is the critical half of
// when a re-teardown finds compute that leaked past the first suspension, it
// must MERGE the newly stopped app into the recorded suspension state, not
// replace it. Otherwise the apps the first teardown stopped are lost from the
// restore map and Resume never brings them back. The merge is observed through
// Resume: with the record merged both apps come back; a replace would drop the
// first app and only the second would resume.
func TestDeploySpendCheck_ReEnforceMergesSuspensionRecord(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	// App 1, running and current.
	dep1 := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})
	setAppCurrent(t, h, dep1.AppID, dep1.ID)

	tEnv := startSpendCheck(t, h.DB)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep1.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	overBudget := func(suspended bool) *hydrav1.CheckWorkspaceSpendRequest {
		return &hydrav1.CheckWorkspaceSpendRequest{
			Period:             period,
			BudgetCents:        1,
			Stop:               true,
			OrgId:              "org_test",
			WorkspaceName:      "test",
			WorkspaceSlug:      "test",
			SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
			CurrentlySuspended: suspended,
		}
	}

	// 1) Suspend: records {app1: dep1} and stops dep1.
	r, err := client.CheckWorkspaceSpend().Request(ctx, overBudget(false))
	require.NoError(t, err)
	require.True(t, r.GetSuspended())
	require.Eventually(t, func() bool {
		app, e := h.DB.FindAppById(ctx, dep1.AppID)
		if e != nil || app.CurrentDeploymentID.Valid {
			return false
		}
		got, e := h.DB.FindDeploymentById(ctx, dep1.ID)
		return e == nil && got.DesiredState == mysqltype.DeploymentsDesiredStateStopped
	}, 15*time.Second, 200*time.Millisecond, "first suspend should stop app1")

	// 2) Leaked compute: a deployment in a second app, created after the first
	//    teardown's snapshot (the gate-read to row-insert race).
	dep2 := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})
	require.NotEqual(t, dep1.AppID, dep2.AppID, "second deployment must be a distinct app")
	setAppCurrent(t, h, dep2.AppID, dep2.ID)

	// 3) Re-enforce: the re-teardown finds dep2 running and MERGES {app2: dep2}
	//    into the recorded {app1: dep1}.
	r, err = client.CheckWorkspaceSpend().Request(ctx, overBudget(true))
	require.NoError(t, err)
	require.True(t, r.GetSuspended())
	require.Eventually(t, func() bool {
		app, e := h.DB.FindAppById(ctx, dep2.AppID)
		if e != nil || app.CurrentDeploymentID.Valid {
			return false
		}
		got, e := h.DB.FindDeploymentById(ctx, dep2.ID)
		return e == nil && got.DesiredState == mysqltype.DeploymentsDesiredStateStopped
	}, 15*time.Second, 200*time.Millisecond, "re-enforcement should stop the leaked app2")

	// 4) Resume with the budget raised above spend. The merged record restores
	//    BOTH apps; a replace bug would have dropped app1.
	r, err = client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
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
	require.False(t, r.GetSuspended(), "budget raised above spend should resume")
	require.Eventually(t, func() bool {
		a1, e1 := h.DB.FindAppById(ctx, dep1.AppID)
		a2, e2 := h.DB.FindAppById(ctx, dep2.AppID)
		if e1 != nil || e2 != nil {
			return false
		}
		return a1.CurrentDeploymentID.Valid && a1.CurrentDeploymentID.String == dep1.ID &&
			a2.CurrentDeploymentID.Valid && a2.CurrentDeploymentID.String == dep2.ID
	}, 15*time.Second, 200*time.Millisecond,
		"merge must restore BOTH apps on resume; a replace would drop app1")
}

// TestDeploySpendCheck_StalePeriodNoOp: an invocation whose request
// period is not the current period (a July tick executing after Aug 1, from a
// cron delay or retries spanning midnight UTC) must enforce nothing. It would
// otherwise suspend a workspace in the new month on the old month's spend.
func TestDeploySpendCheck_StalePeriodNoOp(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})
	setAppCurrent(t, h, dep.AppID, dep.ID)

	tEnv := startSpendCheck(t, h.DB)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)

	// Last month: a valid period, but not the one the journaled Now() falls in.
	stalePeriod := billingperiod.From(time.Now().UTC()).Prev().Key()

	resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
		Period:             stalePeriod,
		BudgetCents:        1,
		Stop:               true,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    200 * deploybilling.MicroCentsPerCent,
		CurrentlySuspended: false,
	})
	require.NoError(t, err)
	require.False(t, resp.GetSuspended(), "a stale-period tick must not suspend")

	// No teardown was dispatched, so compute keeps running.
	require.Never(t, func() bool {
		got, e := h.DB.FindDeploymentById(ctx, dep.ID)
		return e == nil && got.DesiredState == mysqltype.DeploymentsDesiredStateStopped
	}, 2*time.Second, 200*time.Millisecond, "a stale-period tick must not dispatch a teardown")

	billing, err := h.DB.FindWorkspaceBillingByWorkspaceID(ctx, dep.WorkspaceID)
	require.NoError(t, err)
	require.False(t, billing.SpendSuspended, "a stale-period tick must not set the suspended column")
}
