package billingreconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetPlanFeeCents(t *testing.T) {
	t.Run("single plan-fee line: full fee", func(t *testing.T) {
		inv := Invoice{Lines: []InvoiceLine{planFeeLine("plan.pro", "f", 2500, 0)}} //nolint:exhaustruct
		require.Equal(t, int64(2500), netPlanFeeCents(inv))
	})

	t.Run("plan-fee line with its own discount: net of discount", func(t *testing.T) {
		inv := Invoice{Lines: []InvoiceLine{planFeeLine("plan.pro", "f", 2500, 500)}} //nolint:exhaustruct
		require.Equal(t, int64(2000), netPlanFeeCents(inv))
	})

	t.Run("mid-cycle upgrade proration: +new, -unused nets to the top-up", func(t *testing.T) {
		inv := Invoice{Lines: []InvoiceLine{ //nolint:exhaustruct
			planFeeLine("plan.business", "new", 1500, 0),
			planFeeLine("plan.pro", "old", -800, 0),
		}}
		require.Equal(t, int64(700), netPlanFeeCents(inv))
	})

	t.Run("downgrade nets negative: floored at zero", func(t *testing.T) {
		inv := Invoice{Lines: []InvoiceLine{ //nolint:exhaustruct
			planFeeLine("plan.starter", "new", 500, 0),
			planFeeLine("plan.pro", "old", -2000, 0),
		}}
		require.Equal(t, int64(0), netPlanFeeCents(inv))
	})

	t.Run("non-plan lines are ignored", func(t *testing.T) {
		line, _ := meterLineFor(MeterCPUSeconds, 100_000)
		inv := Invoice{Lines: []InvoiceLine{line}} //nolint:exhaustruct
		require.Equal(t, int64(0), netPlanFeeCents(inv))
	})
}

func TestAppliedPromoCreditCents(t *testing.T) {
	inv := Invoice{PretaxCreditAmounts: []PretaxCreditAmount{ //nolint:exhaustruct
		{AmountCents: 1000, Type: PretaxCreditBalanceTransaction},
		{AmountCents: 300, Type: PretaxCreditDiscount}, // unrelated coupon path
		{AmountCents: 200, Type: PretaxCreditBalanceTransaction},
	}}
	require.Equal(t, int64(1200), appliedPromoCreditCents(inv))
}

// creditInvoice builds an invoice applying appliedCents of promo credit.
func creditInvoice(appliedCents int64) Invoice {
	return Invoice{ //nolint:exhaustruct
		ID:                  "in_x",
		PretaxCreditAmounts: []PretaxCreditAmount{{AmountCents: appliedCents, Type: PretaxCreditBalanceTransaction}},
	}
}

// meteredWorth builds a metered map whose single line nets to cents.
func meteredWorth(cents int64) map[Meter]InvoiceLine {
	return map[Meter]InvoiceLine{
		MeterCPUSeconds: {ID: "il", AmountCents: cents, Quantity: 1, PriceID: "p", PriceLookupKey: MeterCPUSeconds.LookupKey(), UnitAmountDecimal: 1, DiscountAmountCents: 0},
	}
}

func TestCheckCredit(t *testing.T) {
	t.Run("applied equals min(grant, metered): clean", func(t *testing.T) {
		findings := checkCredit(creditInvoice(2500), meteredWorth(3000), 2500, true)
		require.Empty(t, findings)
	})

	t.Run("use-it-or-lose-it: grant capped by metered", func(t *testing.T) {
		// grant 2500 but only 196 of metered charge; applying all 196 is clean.
		require.Empty(t, checkCredit(creditInvoice(196), meteredWorth(196), 2500, true))
		// applying the full grant against only 196 of metered is over-applied.
		over := checkCredit(creditInvoice(2500), meteredWorth(196), 2500, true)
		require.Len(t, over, 1)
		require.Equal(t, VerdictStructural, over[0].Class)
	})

	t.Run("short-applied: overbill with positive drift", func(t *testing.T) {
		findings := checkCredit(creditInvoice(2400), meteredWorth(3000), 2500, true)
		require.Len(t, findings, 1)
		require.Equal(t, VerdictOverbill, findings[0].Class)
		require.Equal(t, int64(100), findings[0].DriftCents)
	})

	t.Run("over-applied: structural", func(t *testing.T) {
		findings := checkCredit(creditInvoice(2600), meteredWorth(3000), 2500, true)
		require.Len(t, findings, 1)
		require.Equal(t, VerdictStructural, findings[0].Class)
	})

	t.Run("no previous invoice: no grant, applying none is clean", func(t *testing.T) {
		require.Empty(t, checkCredit(creditInvoice(0), meteredWorth(3000), 0, false))
	})

	t.Run("no previous invoice but credit applied anyway: structural", func(t *testing.T) {
		findings := checkCredit(creditInvoice(50), meteredWorth(3000), 0, false)
		require.Len(t, findings, 1)
		require.Equal(t, VerdictStructural, findings[0].Class)
	})
}

func TestExpectedCreditGrant(t *testing.T) {
	p := fixturePeriod(t)
	priorEnd := p.Start().Unix()
	// The reconciled (matched) invoice's period starts where the funder's ends.
	matched := &InvoiceCandidate{
		ID: "in_cur", Status: InvoiceStatusPaid, BillingReason: "subscription_cycle",
		PeriodStart: p.Start().Unix(), PeriodEnd: p.End().Unix(),
	}

	prev := Invoice{ //nolint:exhaustruct
		ID:          "in_prev",
		Status:      InvoiceStatusPaid,
		PeriodStart: p.Prev().Start().Unix(),
		PeriodEnd:   priorEnd,
		Lines:       []InvoiceLine{planFeeLine("plan.pro", "fee", 2500, 0)},
	}

	newReconciler := func(candidates []InvoiceCandidate, invoices map[string]Invoice) *Reconciler {
		return New(&fakeInvoices{candidates: map[string][]InvoiceCandidate{fixtureSubscription: candidates}, invoices: invoices, findErr: nil, getErr: nil}, nil, nil)
	}

	t.Run("previous paid cycle invoice funds the grant", func(t *testing.T) {
		r := newReconciler([]InvoiceCandidate{candidate(prev)}, map[string]Invoice{prev.ID: prev})
		grant, has, err := r.expectedCreditGrant(context.Background(), []InvoiceCandidate{candidate(prev)}, matched)
		require.NoError(t, err)
		require.True(t, has)
		require.Equal(t, int64(2500), grant)
	})

	t.Run("no previous invoice: hasPrevious false, zero grant", func(t *testing.T) {
		r := newReconciler(nil, nil)
		grant, has, err := r.expectedCreditGrant(context.Background(), nil, matched)
		require.NoError(t, err)
		require.False(t, has)
		require.Zero(t, grant)
	})

	t.Run("previous cycle invoice not paid: not selected", func(t *testing.T) {
		open := prev
		open.Status = InvoiceStatusOpen
		r := newReconciler([]InvoiceCandidate{candidate(open)}, map[string]Invoice{open.ID: open})
		grant, has, err := r.expectedCreditGrant(context.Background(), []InvoiceCandidate{candidate(open)}, matched)
		require.NoError(t, err)
		require.False(t, has)
		require.Zero(t, grant)
	})

	t.Run("a hard read failure propagates", func(t *testing.T) {
		wantErr := errors.New("stripe down")
		r := New(&fakeInvoices{candidates: nil, invoices: nil, findErr: nil, getErr: wantErr}, nil, nil)
		_, _, err := r.expectedCreditGrant(context.Background(), []InvoiceCandidate{candidate(prev)}, matched)
		require.ErrorIs(t, err, wantErr)
	})
}
