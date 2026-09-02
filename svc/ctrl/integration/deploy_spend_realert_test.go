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
	"github.com/unkeyed/unkey/pkg/email"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployspendcheck"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployteardown"
)

// These template aliases mirror the unexported constants in deployspendcheck's
// alert.go; the capturing sender classifies emails by them.
const (
	templateBudgetAlert   = "compute-budget-alert"
	templateBudgetStopped = "compute-budget-stopped"
)

// countWarnings returns how many warning emails were sent for a given budget
// (as the rendered BUDGET string) and percent.
func countWarnings(c *email.Capture, budget, percent string) int {
	n := 0
	for _, e := range c.Sent() {
		if e.TemplateID == templateBudgetAlert &&
			e.Variables["BUDGET"] == budget &&
			e.Variables["PERCENT"] == percent {
			n++
		}
	}
	return n
}

// fixedAdmins resolves one admin email for any org, so the check has a recipient
// to send to (workos.NewNoop resolves none, which would suppress every send).
type fixedAdmins struct{ email string }

func (f fixedAdmins) AdminEmails(_ context.Context, _ string) ([]string, error) {
	return []string{f.email}, nil
}

// startSpendCheckCapturing wires the check with a capturing email sender and a
// resolver that returns one admin, so the tests can assert on emails.
func startSpendCheckCapturing(t *testing.T, database db.Database, sender email.Sender) *restatetest.TestEnvironment {
	t.Helper()

	checkH, err := deployspendcheck.NewCheckHandler(deployspendcheck.CheckConfig{
		DB:             database,
		Admins:         fixedAdmins{email: "admin@example.com"},
		Email:          sender,
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
		deployServiceDefinition(t, database),
		hydrav1.NewDeployTeardownServiceServer(teardownSvc),
		hydrav1.NewDeploySpendCheckServiceServer(checkH),
	)
}

// TestDeploySpendCheck_ReAlertAfterBudgetChange walks the user's reported flow:
// spend hits a $100 budget with stop set (stopped email, suspend), the budget is
// raised to $200 (compute resumes), then spend climbs into the new budget's
// thresholds. The warning at 75% of the raised budget and the second stopped
// email must both fire. On the pre-fix code the raised-budget warning never fires
// because the high-water mark is pinned at 100% from the first suspension, and
// even if it did the idempotency key (period+threshold only) would dedup it
// against the old budget's 75% warning.
func TestDeploySpendCheck_ReAlertAfterBudgetChange(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})

	// Make the deployment its app's current deployment so suspend/resume have
	// something to act on, mirroring the suspend/resume test.
	require.NoError(t, h.DB.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: dep.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
		AppID:               dep.AppID,
	}))

	sender := email.NewCapture()
	tEnv := startSpendCheckCapturing(t, h.DB, sender)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	// grossMicroCents converts a whole-cent gross into micro-cents the way the
	// orchestrator would have priced it.
	grossMicroCents := func(cents int64) int64 { return cents * deploybilling.MicroCentsPerCent }

	suspended := false
	tick := func(budgetCents, grossCents int64, stop bool) {
		t.Helper()
		resp, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
			Period:             period,
			BudgetCents:        budgetCents,
			Stop:               stop,
			OrgId:              "org_test",
			WorkspaceName:      "test",
			WorkspaceSlug:      "test",
			SpendMicroCents:    grossMicroCents(grossCents),
			CurrentlySuspended: suspended,
		})
		require.NoError(t, err)
		suspended = resp.GetSuspended()
	}

	const (
		budgetLow  = 10_000 // $100
		budgetHigh = 20_000 // $200
	)

	// Budget $100, stop set: climb through the thresholds into a suspension.
	tick(budgetLow, 5_000, true)  // 50% of $100 -> 50% warning
	tick(budgetLow, 7_500, true)  // 75% of $100 -> 75% warning
	tick(budgetLow, 10_000, true) // 100% of $100 -> stopped email #1, suspend
	require.True(t, suspended, "spend at budget with stop set should suspend")

	// Raise the budget to $200: spend ($100) is now under it, so compute resumes.
	tick(budgetHigh, 10_000, true) // 50% of $200 -> resume, no email
	require.False(t, suspended, "raising the budget above spend should resume")

	// Spend climbs into the raised budget's thresholds.
	tick(budgetHigh, 15_000, true) // 75% of $200 -> 75% warning at the NEW budget
	tick(budgetHigh, 20_000, true) // 100% of $200 -> stopped email #2, suspend
	require.True(t, suspended, "spend reaching the raised budget with stop set should suspend again")

	// The user's missing email: a 75% warning against the raised $200 budget.
	require.GreaterOrEqual(t, countWarnings(sender, "$200", "75"), 1,
		"a 75%% warning must fire after the budget is raised and re-crossed")

	// Both suspensions must email; the stopped email's idempotency key carries the
	// suspension generation, so the second suspension is not deduped against the
	// first.
	require.Equal(t, 2, sender.CountByTemplate(templateBudgetStopped),
		"each suspension should send a stopped email")

	// Sanity: the original budget's warnings fired too, so this is a re-arm, not a
	// wholesale change of behavior.
	require.Equal(t, 1, countWarnings(sender, "$100", "50"))
	require.Equal(t, 1, countWarnings(sender, "$100", "75"))
}

// TestDeploySpendCheck_BudgetChurnDoesNotSpam guards the anti-spam property: once
// a threshold has been alerted at a spend level, churning the budget (raise,
// lower, raise back) at constant spend must not send another warning. The cron
// ticks every minute, so a design that re-alerted per distinct budget value would
// let a one-cent-per-tick script emit dozens of emails an hour.
func TestDeploySpendCheck_BudgetChurnDoesNotSpam(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})

	sender := email.NewCapture()
	tEnv := startSpendCheckCapturing(t, h.DB, sender)
	client := hydrav1.NewDeploySpendCheckServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	period := time.Now().UTC().Format("2006-01")

	grossMicroCents := func(cents int64) int64 { return cents * deploybilling.MicroCentsPerCent }

	// stop=false throughout so no suspension interferes; spend is fixed at $80.
	tick := func(budgetCents int64) {
		t.Helper()
		_, err := client.CheckWorkspaceSpend().Request(ctx, &hydrav1.CheckWorkspaceSpendRequest{
			Period:             period,
			BudgetCents:        budgetCents,
			Stop:               false,
			OrgId:              "org_test",
			WorkspaceName:      "test",
			WorkspaceSlug:      "test",
			SpendMicroCents:    grossMicroCents(8_000), // $80
			CurrentlySuspended: false,
		})
		require.NoError(t, err)
	}

	tick(10_000) // $80 is 80% of $100 -> one 75% warning
	// Churn the budget without spend moving: none of these may send.
	tick(20_000) // 40% of $200
	tick(10_000) // back to 80% of $100
	tick(20_000) // 40% of $200 again
	tick(10_001) // one-cent nudge, still ~80%
	tick(10_000) // and back

	require.Equal(t, 1, sender.CountByTemplate(templateBudgetAlert),
		"budget churn at constant spend must not re-send the warning")
	require.Equal(t, 1, countWarnings(sender, "$100", "75"))
}

// TestDeploySpendCheck_SuspendedDoesNotWarn guards the torn-state gap: a
// workspace whose suspended column is set but whose high-water mark was never
// recorded (a suspend tick that died after writing the column) must not send a
// stale threshold warning on the next tick. An already-suspended workspace over
// budget stays suspended and emails nothing.
func TestDeploySpendCheck_SuspendedDoesNotWarn(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateStopped,
	})

	// Column says suspended, but the VO carries no high-water state (fresh key),
	// exactly the torn state a killed suspend tick would leave behind.
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
		BudgetCents:        10_000,
		Stop:               true,
		OrgId:              "org_test",
		WorkspaceName:      "test",
		WorkspaceSlug:      "test",
		SpendMicroCents:    20_000 * deploybilling.MicroCentsPerCent, // 200% of budget
		CurrentlySuspended: true,
	})
	require.NoError(t, err)
	require.True(t, resp.GetSuspended(), "an over-budget suspended workspace stays suspended")

	require.Equal(t, 0, sender.CountByTemplate(templateBudgetAlert),
		"a suspended workspace must not receive a threshold warning")
	require.Equal(t, 0, sender.CountByTemplate(templateBudgetStopped),
		"no new suspension transition, so no stopped email")
}
