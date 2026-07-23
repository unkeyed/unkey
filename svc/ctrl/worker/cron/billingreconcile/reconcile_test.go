package billingreconcile

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	reconcile "github.com/unkeyed/unkey/svc/ctrl/internal/billingreconcile"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// fakeEngine is a scripted Engine: it returns the Result (or error) recorded for
// a workspace id, so the orchestration is exercised without a real Stripe or
// ClickHouse.
type fakeEngine struct {
	results map[string]reconcile.Result
	errs    map[string]error
	calls   []reconcile.WorkspaceRef
}

func (f *fakeEngine) ReconcileWorkspace(
	_ context.Context,
	ws reconcile.WorkspaceRef,
	p billingperiod.Period,
) (reconcile.Result, error) {
	f.calls = append(f.calls, ws)
	if err, ok := f.errs[ws.WorkspaceID]; ok {
		return reconcile.Result{}, err //nolint:exhaustruct // error path
	}
	r := f.results[ws.WorkspaceID]
	r.Period = p
	return r, nil
}

func billableRow(id, sub string) db.ListDeployBillableWorkspacesRow {
	return db.ListDeployBillableWorkspacesRow{
		ID:                         id,
		StripeCustomerID:           sql.NullString{Valid: true, String: "cus_" + id},
		StripeDeploySubscriptionID: sql.NullString{Valid: sub != "", String: sub},
	}
}

func TestBuildRefs(t *testing.T) {
	t.Run("keeps rows with a subscription id, skips rows without", func(t *testing.T) {
		rows := []db.ListDeployBillableWorkspacesRow{
			billableRow("ws_a", "sub_a"),
			billableRow("ws_b", ""), // billable but no compute subscription id
			billableRow("ws_c", "sub_c"),
		}

		refs, skipped := buildRefs(rows)

		require.Equal(t, []reconcile.WorkspaceRef{
			{WorkspaceID: "ws_a", StripeSubscriptionID: "sub_a"},
			{WorkspaceID: "ws_c", StripeSubscriptionID: "sub_c"},
		}, refs)
		require.Equal(t, []string{"ws_b"}, skipped)
	})

	t.Run("empty input yields no refs and no skips", func(t *testing.T) {
		refs, skipped := buildRefs(nil)
		require.Empty(t, refs)
		require.Empty(t, skipped)
	})

	t.Run("invalid NullString is treated as no subscription id", func(t *testing.T) {
		rows := []db.ListDeployBillableWorkspacesRow{{
			ID:                         "ws_x",
			StripeCustomerID:           sql.NullString{Valid: true, String: "cus_x"},
			StripeDeploySubscriptionID: sql.NullString{Valid: false, String: ""},
		}}
		refs, skipped := buildRefs(rows)
		require.Empty(t, refs)
		require.Equal(t, []string{"ws_x"}, skipped)
	})
}

func TestSummarize(t *testing.T) {
	results := []reconcile.Result{
		{WorkspaceID: "a", Verdict: reconcile.VerdictClean},
		{WorkspaceID: "b", Verdict: reconcile.VerdictClean},
		{WorkspaceID: "c", Verdict: reconcile.VerdictLateDataUnderbill},
		{WorkspaceID: "d", Verdict: reconcile.VerdictOverbill},
		{WorkspaceID: "e", Verdict: reconcile.VerdictStructural},
		{WorkspaceID: "f", Verdict: reconcile.Verdict("unknown")},
	}

	s := summarize(results, 3)

	require.Equal(t, 2, s.clean)
	require.Equal(t, 1, s.lateDataUnderbill)
	require.Equal(t, 1, s.overbill)
	// Unknown verdicts fold into structural, matching the engine's severity rank.
	require.Equal(t, 2, s.structural)
	require.Equal(t, 3, s.skipped)
}

func TestStructuralResults(t *testing.T) {
	results := []reconcile.Result{
		{WorkspaceID: "a", Verdict: reconcile.VerdictClean},
		{WorkspaceID: "b", Verdict: reconcile.VerdictStructural},
		{WorkspaceID: "c", Verdict: reconcile.VerdictOverbill},
		{WorkspaceID: "d", Verdict: reconcile.VerdictStructural},
	}

	structural := structuralResults(results)

	require.Len(t, structural, 2)
	require.Equal(t, "b", structural[0].WorkspaceID)
	require.Equal(t, "d", structural[1].WorkspaceID)
}

// TestPipeline exercises the enumerate -> reconcile -> classify chain through
// the seams end to end (minus the Restate wiring, which the Handle method owns
// and is compile-verified only): DB rows -> buildRefs -> the fake engine per
// ref -> summarize, proving the pieces compose as the handler wires them.
func TestPipeline(t *testing.T) {
	p := billingperiod.Period{Year: 2026, Month: 6}

	engine := &fakeEngine{
		results: map[string]reconcile.Result{
			"ws_a": {WorkspaceID: "ws_a", InvoiceID: "in_a", Verdict: reconcile.VerdictClean},
			"ws_c": {
				WorkspaceID: "ws_c",
				InvoiceID:   "in_c",
				Verdict:     reconcile.VerdictLateDataUnderbill,
				Findings: []reconcile.Finding{
					{Check: reconcile.CheckQuantity, Class: reconcile.VerdictLateDataUnderbill, Meter: reconcile.MeterCPUSeconds, DriftCents: -60},
				},
			},
		},
		errs:  nil,
		calls: nil,
	}

	rows := []db.ListDeployBillableWorkspacesRow{
		billableRow("ws_a", "sub_a"),
		billableRow("ws_b", ""), // skipped: no subscription id
		billableRow("ws_c", "sub_c"),
	}

	refs, skipped := buildRefs(rows)
	require.Equal(t, []string{"ws_b"}, skipped)

	var results []reconcile.Result
	for _, ref := range refs {
		r, err := engine.ReconcileWorkspace(context.Background(), ref, p)
		require.NoError(t, err)
		results = append(results, r)
	}

	// Every reconcilable workspace was passed to the engine, and the period was
	// threaded through.
	require.Len(t, engine.calls, 2)
	require.Equal(t, p, results[0].Period)

	s := summarize(results, len(skipped))
	require.Equal(t, 1, s.clean)
	require.Equal(t, 1, s.lateDataUnderbill)
	require.Equal(t, 0, s.overbill)
	require.Equal(t, 0, s.structural)
	require.Equal(t, 1, s.skipped)
}
