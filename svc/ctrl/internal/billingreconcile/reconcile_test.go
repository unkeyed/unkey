package billingreconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReconcile_Clean(t *testing.T) {
	f := newCleanFixture(t)
	result := f.run(t)

	require.Equal(t, VerdictClean, result.Verdict)
	require.Empty(t, result.Findings)
	require.Equal(t, fixtureCycleID, result.InvoiceID)
	require.Equal(t, fixtureWorkspaceID, result.WorkspaceID)
	require.Equal(t, f.period, result.Period)
}

func TestReconcile_FirstPeriodNoPreviousInvoiceNoCredit(t *testing.T) {
	// A workspace's first billed period: no renewal invoice, so no grant and no
	// credit expected. The cycle invoice applies none, which is clean.
	f := newCleanFixture(t)
	// Drop the renewal candidate + detail, and the credit it funded.
	f.cycle.PretaxCreditAmounts = nil
	f.cycle.TotalCents = f.cycle.SubtotalCents
	r := f.reconciler()
	fi := r.invoices.(*fakeInvoices)
	fi.candidates[fixtureSubscription] = []InvoiceCandidate{candidate(f.cycle)}
	fi.invoices = map[string]Invoice{f.cycle.ID: f.cycle}

	result, err := r.ReconcileWorkspace(context.Background(),
		WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription}, f.period)
	require.NoError(t, err)
	require.Equal(t, VerdictClean, result.Verdict)
	require.Empty(t, result.Findings)
}

func TestReconcile_Existence_Missing(t *testing.T) {
	f := newCleanFixture(t)
	r := f.reconciler()
	// Only the previous renewal remains; the current cycle invoice is gone.
	fi := r.invoices.(*fakeInvoices)
	fi.candidates[fixtureSubscription] = []InvoiceCandidate{candidate(f.renewal)}

	result, err := r.ReconcileWorkspace(context.Background(),
		WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription}, f.period)
	require.NoError(t, err)
	require.Equal(t, VerdictStructural, result.Verdict)
	require.Empty(t, result.InvoiceID)
	require.Len(t, result.Findings, 1)
	require.Equal(t, CheckExistence, result.Findings[0].Check)
	require.Contains(t, result.Findings[0].Detail, "no subscription_cycle invoice")
}

func TestReconcile_Existence_PreviousCycleIsNotAFalseDuplicate(t *testing.T) {
	// The renewal invoice ends exactly at this period's start; the half-open
	// overlap must exclude it so it is never counted as a duplicate or a
	// wrong-window invoice for this period. With no current cycle invoice it
	// reads as missing, not duplicate.
	f := newCleanFixture(t)
	r := f.reconciler()
	fi := r.invoices.(*fakeInvoices)
	fi.candidates[fixtureSubscription] = []InvoiceCandidate{candidate(f.renewal)}

	result, err := r.ReconcileWorkspace(context.Background(),
		WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription}, f.period)
	require.NoError(t, err)
	require.Equal(t, VerdictStructural, result.Verdict)
	require.Contains(t, result.Findings[0].Detail, "no subscription_cycle invoice")
}

func TestReconcile_Existence_Duplicate(t *testing.T) {
	f := newCleanFixture(t)
	dup := f.cycle
	dup.ID = "in_cycle_dup"
	r := f.reconciler()
	fi := r.invoices.(*fakeInvoices)
	fi.candidates[fixtureSubscription] = append(fi.candidates[fixtureSubscription], candidate(dup))
	fi.invoices[dup.ID] = dup

	result, err := r.ReconcileWorkspace(context.Background(),
		WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription}, f.period)
	require.NoError(t, err)
	require.Equal(t, VerdictStructural, result.Verdict)
	require.Empty(t, result.InvoiceID)
	require.Len(t, result.Findings, 1)
	require.Contains(t, result.Findings[0].Detail, "2 finalized subscription_cycle invoices")
	require.Contains(t, result.Findings[0].Detail, "in_cycle_dup")
}

func TestReconcile_Existence_WrongWindow(t *testing.T) {
	// Anchor collapse: the invoice bills a window that overlaps June but is
	// shifted 14 days, so it is not the calendar month.
	f := newCleanFixture(t)
	f.cycle.PeriodStart = f.period.Start().AddDate(0, 0, 14).Unix()
	f.cycle.PeriodEnd = f.period.End().AddDate(0, 0, 14).Unix()
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	require.Empty(t, result.InvoiceID)
	require.Len(t, result.Findings, 1)
	require.Contains(t, result.Findings[0].Detail, "does not end on the calendar-month boundary")
}

func TestReconcile_Existence_NotFinalized(t *testing.T) {
	f := newCleanFixture(t)
	f.cycle.Status = InvoiceStatusDraft
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	require.Empty(t, result.InvoiceID)
	require.Len(t, result.Findings, 1)
	require.Contains(t, result.Findings[0].Detail, "never finalized")
}

func TestReconcile_Existence_VoidIsMissing(t *testing.T) {
	// A void invoice was finalized then reversed; it is not a real charge, so
	// the period reads as missing (a draft would read as not-finalized).
	f := newCleanFixture(t)
	f.cycle.Status = InvoiceStatusVoid
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	require.Contains(t, result.Findings[0].Detail, "no subscription_cycle invoice")
}

func TestReconcile_LateDataUnderbill(t *testing.T) {
	f := newCleanFixture(t)
	// ClickHouse settled 1,000,000 more CPU-seconds than were billed: ~694
	// cents of shortfall, well past the materiality bar.
	f.usage.CPUSeconds += 1_000_000
	result := f.run(t)

	require.Equal(t, VerdictLateDataUnderbill, result.Verdict)
	q := findingsFor(result, CheckQuantity)
	require.Len(t, q, 1)
	require.Equal(t, MeterCPUSeconds, q[0].Meter)
	require.Less(t, q[0].DriftCents, int64(0))
}

func TestReconcile_LateDataBelowBarIsClean(t *testing.T) {
	f := newCleanFixture(t)
	// +100 active keys is 20 cents, under the $0.50 floor.
	f.usage.ActiveKeys += 100
	result := f.run(t)

	require.Equal(t, VerdictClean, result.Verdict)
	require.Empty(t, result.Findings)
}

func TestReconcile_Overbill(t *testing.T) {
	f := newCleanFixture(t)
	// ClickHouse recorded far less CPU than was billed: impossible under the
	// convergent push, so it is an overbill.
	f.usage.CPUSeconds = 10_000
	result := f.run(t)

	require.Equal(t, VerdictOverbill, result.Verdict)
	q := findingsFor(result, CheckQuantity)
	require.Len(t, q, 1)
	require.Equal(t, MeterCPUSeconds, q[0].Meter)
	require.Greater(t, q[0].DriftCents, int64(0))
}

func TestReconcile_Overbill_NoDollarFloorButAboveOneCent(t *testing.T) {
	// A gap that prices above 1 cent pages regardless of size; the invoice-higher
	// direction gets no dollar materiality bar.
	f := newCleanFixture(t)
	// Drop ~10 cents of egress (2 GiB * 5c) from the recorded side.
	f.usage.EgressGiB -= 2
	result := f.run(t)

	require.Equal(t, VerdictOverbill, result.Verdict)
	q := findingsFor(result, CheckQuantity)
	require.Len(t, q, 1)
	require.Equal(t, MeterEgressGiB, q[0].Meter)
	require.Equal(t, int64(10), q[0].DriftCents)
}

func TestReconcile_PriceRateDivergence(t *testing.T) {
	f := newCleanFixture(t)
	p := f.prices[MeterEgressGiB.LookupKey()]
	p.UnitAmountDecimal *= 1.1 // Stripe-side catalog drift
	f.prices[MeterEgressGiB.LookupKey()] = p
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	price := findingsFor(result, CheckPrice)
	require.Len(t, price, 1)
	require.Equal(t, MeterEgressGiB, price[0].Meter)
	require.Contains(t, price[0].Detail, "diverges from pinned catalog rate")
}

func TestReconcile_StalePriceOnSubscription(t *testing.T) {
	// The lookup_key was repriced to a new price object, but the subscription
	// still bills through the old price id. Structural even though the rate
	// still matches.
	f := newCleanFixture(t)
	moved := f.prices[MeterCPUSeconds.LookupKey()]
	moved.ID = "price_cpu_seconds_v2"
	f.prices[MeterCPUSeconds.LookupKey()] = moved
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	price := findingsFor(result, CheckPrice)
	require.Len(t, price, 1)
	require.Contains(t, price[0].Detail, "resolves to price_cpu_seconds_v2")
}

func TestReconcile_PriceMissing(t *testing.T) {
	f := newCleanFixture(t)
	delete(f.prices, MeterDiskGiBSeconds.LookupKey())
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	price := findingsFor(result, CheckPrice)
	require.Len(t, price, 1)
	require.Contains(t, price[0].Detail, "no stripe price")
}

func TestReconcile_ShortAppliedCreditIsOverbill(t *testing.T) {
	f := newCleanFixture(t)
	// Applied 100 cents less credit than the grant funds; keep the total
	// consistent so only the credit check fires.
	f.cycle.PretaxCreditAmounts[0].AmountCents -= 100
	f.cycle.TotalCents += 100
	result := f.run(t)

	require.Equal(t, VerdictOverbill, result.Verdict)
	credit := findingsFor(result, CheckCredit)
	require.Len(t, credit, 1)
	require.Equal(t, VerdictOverbill, credit[0].Class)
	require.Equal(t, int64(100), credit[0].DriftCents)
}

func TestReconcile_OverAppliedCreditIsStructural(t *testing.T) {
	f := newCleanFixture(t)
	f.cycle.PretaxCreditAmounts[0].AmountCents += 100
	f.cycle.TotalCents -= 100
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	credit := findingsFor(result, CheckCredit)
	require.Len(t, credit, 1)
	require.Equal(t, VerdictStructural, credit[0].Class)
	require.Contains(t, credit[0].Detail, "more than")
}

func TestReconcile_UnpaidPreviousFeeFundsNoCredit(t *testing.T) {
	// The renewal fee invoice was never paid, so no grant was created; zero
	// applied credit is correct, not an overbill.
	f := newCleanFixture(t)
	f.renewal.Status = InvoiceStatusOpen
	f.cycle.PretaxCreditAmounts = nil
	f.cycle.TotalCents = f.cycle.SubtotalCents
	r := f.reconciler()
	fi := r.invoices.(*fakeInvoices)
	fi.candidates[fixtureSubscription] = []InvoiceCandidate{candidate(f.renewal), candidate(f.cycle)}
	fi.invoices[f.renewal.ID] = f.renewal
	fi.invoices[f.cycle.ID] = f.cycle

	result, err := r.ReconcileWorkspace(context.Background(),
		WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription}, f.period)
	require.NoError(t, err)
	require.Equal(t, VerdictClean, result.Verdict)
	require.Empty(t, result.Findings)
}

func TestReconcile_TotalDecompositionFailure(t *testing.T) {
	f := newCleanFixture(t)
	f.cycle.TotalCents++ // an unexplained cent
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	total := findingsFor(result, CheckTotal)
	require.Len(t, total, 1)
	require.Contains(t, total[0].Detail, "unexplained")
}

func TestReconcile_UnrecognizedLineIsStructural(t *testing.T) {
	f := newCleanFixture(t)
	f.cycle.Lines = append(f.cycle.Lines, InvoiceLine{
		ID:                  "il_mystery",
		AmountCents:         100,
		Quantity:            1,
		PriceID:             "price_unknown",
		PriceLookupKey:      "",
		UnitAmountDecimal:   100,
		DiscountAmountCents: 0,
	})
	f.cycle.SubtotalCents += 100
	f.cycle.TotalCents += 100
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	var found bool
	for _, fnd := range result.Findings {
		if fnd.Check == CheckExistence && fnd.Meter == "" {
			require.Contains(t, fnd.Detail, "unrecognized price")
			found = true
		}
	}
	require.True(t, found)
}

func TestReconcile_DuplicateMeterLineIsStructural(t *testing.T) {
	f := newCleanFixture(t)
	dupLine, _ := meterLineFor(MeterEgressGiB, meterQuantities(f.usage)[MeterEgressGiB])
	dupLine.ID = "il_egress_dup"
	f.cycle.Lines = append(f.cycle.Lines, dupLine)
	f.cycle.SubtotalCents += dupLine.AmountCents
	// Keep the credit in sync: the duplicated meter is dropped from the metered
	// set, so metered net falls by that line; grant still exceeds it, so applied
	// credit stays the grant. Only the duplicate-line finding should fire.
	f.cycle.TotalCents += dupLine.AmountCents
	result := f.run(t)

	require.Equal(t, VerdictStructural, result.Verdict)
	var duplicate bool
	for _, fnd := range result.Findings {
		if fnd.Meter == MeterEgressGiB && fnd.Check == CheckExistence {
			require.Contains(t, fnd.Detail, "multiple lines")
			duplicate = true
		}
	}
	require.True(t, duplicate)
}

func TestReconcile_VerdictPrecedence(t *testing.T) {
	t.Run("overbill outranks late_data_underbill", func(t *testing.T) {
		f := newCleanFixture(t)
		f.usage.CPUSeconds = 10_000       // overbill on cpu
		f.usage.DiskGiBSeconds += 200_000 // 120 cents late data on disk, past the bar
		result := f.run(t)
		require.Equal(t, VerdictOverbill, result.Verdict)
	})

	t.Run("structural outranks overbill", func(t *testing.T) {
		f := newCleanFixture(t)
		f.usage.CPUSeconds = 10_000 // overbill
		f.cycle.TotalCents++        // structural
		result := f.run(t)
		require.Equal(t, VerdictStructural, result.Verdict)
	})
}

func TestReconcile_EmptyWorkspaceRefIsError(t *testing.T) {
	f := newCleanFixture(t)
	_, err := f.reconciler().ReconcileWorkspace(context.Background(),
		WorkspaceRef{WorkspaceID: "", StripeSubscriptionID: ""}, f.period)
	require.Error(t, err)
}

func TestReconcile_FailureHandling(t *testing.T) {
	wantErr := errors.New("upstream is down")
	ws := WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription}

	t.Run("list error aborts with a hard error", func(t *testing.T) {
		f := newCleanFixture(t)
		r := f.reconciler()
		r.invoices.(*fakeInvoices).findErr = wantErr
		_, err := r.ReconcileWorkspace(context.Background(), ws, f.period)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("get invoice error aborts with a hard error", func(t *testing.T) {
		f := newCleanFixture(t)
		r := f.reconciler()
		r.invoices.(*fakeInvoices).getErr = wantErr
		_, err := r.ReconcileWorkspace(context.Background(), ws, f.period)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("usage error aborts with a hard error", func(t *testing.T) {
		f := newCleanFixture(t)
		r := f.reconciler()
		r.usage.(*fakeUsage).err = wantErr
		_, err := r.ReconcileWorkspace(context.Background(), ws, f.period)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("price read error aborts, never folds into a structural verdict", func(t *testing.T) {
		f := newCleanFixture(t)
		r := f.reconciler()
		r.prices.(*fakePrices).err = wantErr
		_, err := r.ReconcileWorkspace(context.Background(), ws, f.period)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("price ErrNotFound folds into the result, not an error", func(t *testing.T) {
		f := newCleanFixture(t)
		delete(f.prices, MeterCPUSeconds.LookupKey())
		result := f.run(t)
		require.Equal(t, VerdictStructural, result.Verdict)
	})
}
