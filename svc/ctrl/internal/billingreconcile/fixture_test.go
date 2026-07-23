package billingreconcile

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

const (
	fixtureWorkspaceID   = "ws_1"
	fixtureSubscription  = "sub_1"
	fixtureCycleID       = "in_cycle"
	fixtureRenewalID     = "in_renewal"
	fixturePlanLookupKey = "plan.pro"
	fixturePlanPriceID   = "price_plan_pro"
	fixturePlanFeeCents  = int64(2500) // $25/mo, mirrors tools/pricing's "pro" plan
)

// fixturePeriod is the calendar month every test reconciles against.
func fixturePeriod(t *testing.T) billingperiod.Period {
	t.Helper()
	p, err := billingperiod.Parse("2026-06")
	require.NoError(t, err)
	return p
}

// fixtureUsage is the live ClickHouse usage the clean fixture starts from:
// distinct, non-round per-meter values (so a bug swapping two meters shows as a
// mismatch) sized so the net metered charge exceeds the plan fee, keeping
// min(grant, metered) == grant.
func fixtureUsage() billingmeter.MeterValues {
	return billingmeter.MeterValues{
		CPUSeconds:       1_000_000, // 694 cents
		MemoryGiBSeconds: 3_000_000, // 1042 cents
		EgressGiB:        200,       // 1000 cents
		DiskGiBSeconds:   4_000_000, // 24 cents
		ActiveKeys:       250,       // 50 cents
	}
}

func meterPriceID(m Meter) string { return "price_" + string(m) }

// meterLineFor builds the invoice line and matching catalog price for one meter
// at the given billed quantity. Amount is computed exactly as Stripe would:
// round(quantity * rate).
func meterLineFor(m Meter, qty float64) (InvoiceLine, Price) {
	rate := rateCents(m)
	line := InvoiceLine{
		ID:                  "il_" + string(m),
		AmountCents:         int64(math.Round(qty * rate)),
		Quantity:            qty,
		PriceID:             meterPriceID(m),
		PriceLookupKey:      m.LookupKey(),
		UnitAmountDecimal:   rate,
		DiscountAmountCents: 0,
	}
	price := Price{ID: meterPriceID(m), LookupKey: m.LookupKey(), UnitAmountDecimal: rate}
	return line, price
}

// planFeeLine builds a plan-fee line at the given gross amount and discount.
func planFeeLine(lookupKey, id string, amountCents, discountCents int64) InvoiceLine {
	return InvoiceLine{
		ID:                  "il_" + id,
		AmountCents:         amountCents,
		Quantity:            1,
		PriceID:             fixturePlanPriceID,
		PriceLookupKey:      lookupKey,
		UnitAmountDecimal:   float64(amountCents),
		DiscountAmountCents: discountCents,
	}
}

// fixture is a fully self-consistent "everything matches" scenario. Every test
// mutates one thing off it.
type fixture struct {
	period  billingperiod.Period
	usage   billingmeter.MeterValues
	cycle   Invoice
	renewal Invoice
	prices  map[string]Price
}

// newCleanFixture builds the clean scenario: the reconciled cycle invoice with
// one line per meter matching live usage and next period's plan fee billed in
// advance, the previous renewal invoice whose plan fee funds this period's
// credit, the applied credit at exactly min(grant, metered) == grant, and a
// total that decomposes.
func newCleanFixture(t *testing.T) fixture {
	t.Helper()
	period := fixturePeriod(t)
	usage := fixtureUsage()
	live := meterQuantities(usage)

	lines := make([]InvoiceLine, 0, len(Meters())+1)
	prices := make(map[string]Price, len(Meters()))
	var subtotal, meteredNet int64
	for _, m := range Meters() {
		line, price := meterLineFor(m, live[m])
		lines = append(lines, line)
		prices[m.LookupKey()] = price
		subtotal += line.AmountCents
		meteredNet += line.AmountCents
	}

	// Next period's plan fee, billed in advance on this invoice; funds NEXT
	// period's credit, not this one's.
	nextFee := planFeeLine(fixturePlanLookupKey, "fee_next", fixturePlanFeeCents, 0)
	lines = append(lines, nextFee)
	subtotal += nextFee.AmountCents

	applied := fixturePlanFeeCents // grant < metered, so all of the grant applies
	require.Greater(t, meteredNet, fixturePlanFeeCents, "fixture must have metered charge above the plan fee")

	cycle := Invoice{
		ID:                  fixtureCycleID,
		Status:              InvoiceStatusPaid,
		BillingReason:       "subscription_cycle",
		PeriodStart:         period.Start().Unix(),
		PeriodEnd:           period.End().Unix(),
		SubtotalCents:       subtotal,
		TotalCents:          subtotal - applied,
		AmountShippingCents: 0,
		DiscountAmounts:     nil,
		PretaxCreditAmounts: []PretaxCreditAmount{{AmountCents: applied, Type: PretaxCreditBalanceTransaction}},
		Taxes:               nil,
		Lines:               lines,
	}

	// The previous cycle invoice: its plan-fee line covers this period and funds
	// this period's credit. Its own metered lines are for the prior period and
	// are irrelevant to the grant, so they are omitted.
	renewalFee := planFeeLine(fixturePlanLookupKey, "fee_current", fixturePlanFeeCents, 0)
	renewal := Invoice{
		ID:                  fixtureRenewalID,
		Status:              InvoiceStatusPaid,
		BillingReason:       "subscription_cycle",
		PeriodStart:         period.Prev().Start().Unix(),
		PeriodEnd:           period.Start().Unix(),
		SubtotalCents:       fixturePlanFeeCents,
		TotalCents:          fixturePlanFeeCents,
		AmountShippingCents: 0,
		DiscountAmounts:     nil,
		PretaxCreditAmounts: nil,
		Taxes:               nil,
		Lines:               []InvoiceLine{renewalFee},
	}

	return fixture{period: period, usage: usage, cycle: cycle, renewal: renewal, prices: prices}
}

// candidate reduces an invoice to its coarse existence view.
func candidate(inv Invoice) InvoiceCandidate {
	return InvoiceCandidate{ID: inv.ID, Status: inv.Status, BillingReason: inv.BillingReason, PeriodStart: inv.PeriodStart, PeriodEnd: inv.PeriodEnd}
}

// reconciler wires the fixture into a Reconciler over fakes.
func (f fixture) reconciler() *Reconciler {
	invoices := &fakeInvoices{
		candidates: map[string][]InvoiceCandidate{
			fixtureSubscription: {candidate(f.renewal), candidate(f.cycle)},
		},
		invoices: map[string]Invoice{
			f.cycle.ID:   f.cycle,
			f.renewal.ID: f.renewal,
		},
		findErr: nil,
		getErr:  nil,
	}
	return New(
		invoices,
		&fakePrices{byLookupKey: f.prices, err: nil},
		&fakeUsage{byWorkspace: map[string]billingmeter.MeterValues{fixtureWorkspaceID: f.usage}, err: nil},
	)
}

func (f fixture) run(t *testing.T) Result {
	t.Helper()
	result, err := f.reconciler().ReconcileWorkspace(
		context.Background(),
		WorkspaceRef{WorkspaceID: fixtureWorkspaceID, StripeSubscriptionID: fixtureSubscription},
		f.period,
	)
	require.NoError(t, err)
	return result
}

// findingsFor returns the findings of one check.
func findingsFor(result Result, check Check) []Finding {
	var out []Finding
	for _, f := range result.Findings {
		if f.Check == check {
			out = append(out, f)
		}
	}
	return out
}
