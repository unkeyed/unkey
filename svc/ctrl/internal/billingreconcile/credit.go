package billingreconcile

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/fault"
)

// expectedCreditGrant sizes the credit this period is entitled to from the paid
// plan-fee invoice ending where the matched invoice starts: the previous cycle
// in steady state, or the subscription_create proration for a first cycle. Paid
// only (the grant is minted on payment). hasPrevious=false means nothing funds
// this period, so no credit is expected.
func (r *Reconciler) expectedCreditGrant(ctx context.Context, candidates []InvoiceCandidate, matched *InvoiceCandidate) (int64, bool, error) {
	var funderID string
	for _, c := range candidates {
		if c.Status == InvoiceStatusPaid && c.ID != matched.ID && c.PeriodEnd == matched.PeriodStart {
			funderID = c.ID
			break
		}
	}
	if funderID == "" {
		return 0, false, nil
	}

	funder, err := r.invoices.GetInvoice(ctx, funderID)
	if err != nil {
		return 0, false, fault.Wrap(err, fault.Internal("get funding invoice "+funderID))
	}
	return netPlanFeeCents(funder), true, nil
}

// netPlanFeeCents sums the invoice's plan-fee lines net of discounts, floored at
// zero (mirrors netDeployFee in deployCredits.ts). Summing the invoiced lines
// keeps upgrades/downgrades correct without reading a flat-fee constant.
func netPlanFeeCents(inv Invoice) int64 {
	var net int64
	for _, line := range inv.Lines {
		if !strings.HasPrefix(line.PriceLookupKey, planLookupPrefix) {
			continue
		}
		net += line.AmountCents - line.DiscountAmountCents
	}
	if net < 0 {
		return 0
	}
	return net
}

// checkCredit compares applied credit against the expected min(grant, net
// metered): under-applied overcharges the customer (overbill); over-applied is a
// grant-path bug (structural).
func checkCredit(inv Invoice, metered map[Meter]InvoiceLine, grantCents int64, hasPrevious bool) []Finding {
	applied := appliedPromoCreditCents(inv)

	var meteredNet int64
	for _, line := range metered {
		meteredNet += line.AmountCents - line.DiscountAmountCents
	}
	if meteredNet < 0 {
		meteredNet = 0
	}

	expectedApplied := grantCents
	if meteredNet < expectedApplied {
		expectedApplied = meteredNet
	}

	switch {
	case applied < expectedApplied:
		return []Finding{{
			Check: CheckCredit, Class: VerdictOverbill, Meter: "", DriftCents: expectedApplied - applied,
			Detail: fmt.Sprintf("invoice %s applied %d cents of credit but the prior fee entitles %d (grant %d, metered %d, previous invoice %t)",
				inv.ID, applied, expectedApplied, grantCents, meteredNet, hasPrevious),
		}}
	case applied > expectedApplied:
		return []Finding{{
			Check: CheckCredit, Class: VerdictStructural, Meter: "", DriftCents: expectedApplied - applied,
			Detail: fmt.Sprintf("invoice %s applied %d cents of credit, more than the %d its prior fee funds (grant %d, metered %d, previous invoice %t)",
				inv.ID, applied, expectedApplied, grantCents, meteredNet, hasPrevious),
		}}
	}
	return nil
}

// appliedPromoCreditCents sums the credit_balance_transaction pretax credits
// (where billing credit grants land); "discount" credits are a separate path.
func appliedPromoCreditCents(inv Invoice) int64 {
	var sum int64
	for _, c := range inv.PretaxCreditAmounts {
		if c.Type == PretaxCreditBalanceTransaction {
			sum += c.AmountCents
		}
	}
	return sum
}
