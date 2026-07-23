package billingreconcile

import "fmt"

// checkTotal decomposes the invoice total exhaustively, with no tolerance:
// every term is an integer cent Stripe itself computed, so an exact sum either
// holds or something is unaccounted for. Two independent equalities must hold:
// the subtotal is exactly the sum of line amounts (nothing adjusts a line
// outside its own amount), and subtotal minus discounts minus applied credit
// grants plus tax plus shipping equals the total. An unexplained cent or an
// adjustment type this package does not recognize is a code gap, not a
// billed-amount decision, so it is structural.
func checkTotal(inv Invoice) []Finding {
	var findings []Finding

	var linesSum int64
	for _, line := range inv.Lines {
		linesSum += line.AmountCents
	}
	if linesSum != inv.SubtotalCents {
		findings = append(findings, Finding{
			Check: CheckTotal, Class: VerdictStructural, Meter: "", DriftCents: inv.SubtotalCents - linesSum,
			Detail: fmt.Sprintf("invoice %s subtotal %d cents but lines sum to %d",
				inv.ID, inv.SubtotalCents, linesSum),
		})
	}

	expected := inv.SubtotalCents
	for _, d := range inv.DiscountAmounts {
		expected -= d.AmountCents
	}
	for _, c := range inv.PretaxCreditAmounts {
		switch c.Type {
		case PretaxCreditBalanceTransaction:
			// An applied credit grant reduces the total.
			expected -= c.AmountCents
		case PretaxCreditDiscount:
			// A coupon mirror of total_discount_amounts, already subtracted
			// above; the closing equation catches any disagreement.
		default:
			findings = append(findings, Finding{
				Check: CheckTotal, Class: VerdictStructural, Meter: "", DriftCents: c.AmountCents,
				Detail: fmt.Sprintf("invoice %s carries unrecognized adjustment type %q (%d cents)",
					inv.ID, c.Type, c.AmountCents),
			})
		}
	}
	expected += inv.AmountShippingCents
	for _, t := range inv.Taxes {
		expected += t.AmountCents
	}

	if expected != inv.TotalCents {
		findings = append(findings, Finding{
			Check: CheckTotal, Class: VerdictStructural, Meter: "", DriftCents: inv.TotalCents - expected,
			Detail: fmt.Sprintf("invoice %s total %d cents unexplained: decomposition yields %d",
				inv.ID, inv.TotalCents, expected),
		})
	}
	return findings
}
