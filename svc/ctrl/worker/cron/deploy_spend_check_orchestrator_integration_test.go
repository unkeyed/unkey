package cron_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
)

func seedBudgetedWorkspace(
	t *testing.T,
	h *harness.Harness,
	customerID string,
	budgetCents int64,
) string {
	t.Helper()
	ws := h.Seed.CreateWorkspace(h.Ctx)
	_, err := h.DB.RW().ExecContext(
		h.Ctx,
		`UPDATE workspace_billing SET
			plan = ?,
			stripe_customer_id = ?,
			spend_budget_cents = ?,
			spend_budget_stop = ?
		WHERE workspace_id = ?`,
		"pro", customerID, budgetCents, true, ws.ID,
	)
	require.NoError(t, err)
	clearBudgetOnCleanup(t, h, ws.ID)
	return ws.ID
}

// clearBudgetOnCleanup removes the workspace from the spend-check opt-in set
// when the subtest ends. The orchestrator counts are asserted exactly, so a
// budgeted workspace leaking into a later subtest skews its counters.
func clearBudgetOnCleanup(t *testing.T, h *harness.Harness, workspaceID string) {
	t.Helper()
	t.Cleanup(func() {
		_, err := h.DB.RW().ExecContext(h.Ctx,
			`UPDATE workspace_billing SET spend_budget_cents = NULL WHERE workspace_id = ?`, workspaceID)
		require.NoError(t, err)
	})
}

// TestRunDeploySpendCheck_OrchestratorIntegration exercises the fleet scan and
// fan-out decision without driving the per-workspace check to completion.
func TestRunDeploySpendCheck_OrchestratorIntegration(t *testing.T) {
	reader := &fakeUsageReader{} //nolint:exhaustruct // set per subtest
	h := harness.New(t, harness.WithDeployBilling(reader, newFakePusher(), newFakeCloser()))

	// The harness database persists across test runs, and earlier runs can leave
	// budgeted or spend-suspended workspaces behind. Both are in the
	// orchestrator's dispatch set, so clear both before exact count assertions.
	_, err := h.DB.RW().ExecContext(h.Ctx,
		`UPDATE workspace_billing
		SET spend_budget_cents = NULL, spend_suspended = FALSE
		WHERE spend_budget_cents IS NOT NULL OR spend_suspended = TRUE`)
	require.NoError(t, err)

	period := time.Now().UTC().Format("2006-01")
	run := func() (*hydrav1.RunDeploySpendCheckResponse, error) {
		return hydrav1.NewCronServiceIngressClient(h.Restate, "deploy-spend-check-"+period).
			RunDeploySpendCheck().
			Request(h.Ctx, &hydrav1.RunDeploySpendCheckRequest{})
	}

	t.Run("dispatches workspace at or above 50% threshold", func(t *testing.T) {
		reader.set(nil)
		customerID := uid.New("cus")
		wsID := seedBudgetedWorkspace(t, h, customerID, 100)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 100_000},
		})

		resp, err := run()
		require.NoError(t, err)
		require.Equal(t, int32(1), resp.GetWorkspacesDispatched())
	})

	t.Run("skips quiet workspace below lowest alert threshold", func(t *testing.T) {
		reader.set(nil)
		customerID := uid.New("cus")
		wsID := seedBudgetedWorkspace(t, h, customerID, 1_000_000)
		reader.set([]clickhouse.InstanceMeterUsage{
			{WorkspaceID: wsID, ResourceID: "r1", CPUSeconds: 1},
		})

		resp, err := run()
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.GetWorkspacesDispatched())
	})

	t.Run("limits concurrent instance usage shards", func(t *testing.T) {
		reader.set(nil)
		for range 8 {
			seedBudgetedWorkspace(t, h, uid.New("cus"), 1_000_000)
		}
		reader.trackInstanceConcurrency(50 * time.Millisecond)

		resp, err := run()
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.GetWorkspacesDispatched())
		instanceReads, _ := reader.reads()
		require.Equal(t, 8, instanceReads)
		require.Equal(t, 2, reader.maxInstanceConcurrency())
	})

	t.Run("failed usage read does not wedge the period object", func(t *testing.T) {
		reader.set(nil)
		seedBudgetedWorkspace(t, h, uid.New("cus"), 1_000_000)
		reader.failInstanceReads(errors.New("forced usage read failure"))

		started := time.Now()
		_, err := run()
		require.ErrorContains(t, err, "forced usage read failure")
		require.Less(t, time.Since(started), 10*time.Second)
		instanceReads, _ := reader.reads()
		require.Equal(t, 5, instanceReads)

		reader.set(nil)
		resp, err := run()
		require.NoError(t, err)
		require.Equal(t, int32(0), resp.GetWorkspacesDispatched())
	})
}
