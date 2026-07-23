package billingreconcile

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/unkeyed/unkey/pkg/fault"
)

// rateRelativeTolerance bounds how far a rate may sit from the pinned catalog
// rate and still match. Both sides parse the same decimal literal text (a Go
// source constant, a JSON decimal string) so they are expected to land on the
// same nearest float64; the tolerance absorbs any last-bit parse wobble rather
// than depending on that coincidence.
const rateRelativeTolerance = 1e-9

// lineAmountToleranceCents allows Stripe's per-line rounding when the pinned
// rate is used to recompute quantity x rate against the billed line amount.
const lineAmountToleranceCents = 1

// checkPrices verifies, for every metered line, that (a) the price Stripe
// resolves for the meter's lookup_key is the price object the line billed
// through, (b) that price's unit rate and the line's own unit rate both equal
// the pinned catalog rate, and (c) the pinned rate reproduces the line amount
// within one cent of Stripe's per-line rounding. Any divergence is structural
// (catalog or code drift, not a money decision). Meters with no billed line are
// skipped: there is no price object to compare and no amount to recompute, and
// any drift on a rate we never billed this period surfaces on a workspace that
// did. A genuine read failure returns an error; ErrNotFound (no price for the
// lookup_key) is folded into a structural finding.
func (r *Reconciler) checkPrices(ctx context.Context, metered map[Meter]InvoiceLine) ([]Finding, error) {
	var findings []Finding
	for _, m := range Meters() {
		line, ok := metered[m]
		if !ok {
			continue
		}
		pinned := rateCents(m)

		price, err := r.prices.PriceByLookupKey(ctx, m.LookupKey())
		if errors.Is(err, ErrNotFound) {
			findings = append(findings, Finding{
				Check: CheckPrice, Class: VerdictStructural, Meter: m, DriftCents: 0,
				Detail: fmt.Sprintf("meter %s has no stripe price for lookup key %s", m, m.LookupKey()),
			})
			continue
		}
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("fetch stripe price "+m.LookupKey()))
		}

		if price.ID != line.PriceID {
			findings = append(findings, Finding{
				Check: CheckPrice, Class: VerdictStructural, Meter: m, DriftCents: 0,
				Detail: fmt.Sprintf("meter %s bills through price %s but lookup key %s resolves to %s",
					m, line.PriceID, m.LookupKey(), price.ID),
			})
		}
		if !ratesEqual(price.UnitAmountDecimal, pinned) {
			findings = append(findings, Finding{
				Check: CheckPrice, Class: VerdictStructural, Meter: m, DriftCents: 0,
				Detail: fmt.Sprintf("meter %s stripe rate %.10f cents diverges from pinned catalog rate %.10f",
					m, price.UnitAmountDecimal, pinned),
			})
		}
		if !ratesEqual(line.UnitAmountDecimal, pinned) {
			findings = append(findings, Finding{
				Check: CheckPrice, Class: VerdictStructural, Meter: m, DriftCents: 0,
				Detail: fmt.Sprintf("meter %s line rate %.10f cents diverges from pinned catalog rate %.10f",
					m, line.UnitAmountDecimal, pinned),
			})
		}

		// Stripe computes the line amount as quantity x unit_amount_decimal
		// rounded to the nearest cent (half away from zero, which Go's
		// math.Round matches); allow one cent for that rounding.
		recomputed := int64(math.Round(line.Quantity * pinned))
		if absInt64(recomputed-line.AmountCents) > lineAmountToleranceCents {
			findings = append(findings, Finding{
				Check: CheckPrice, Class: VerdictStructural, Meter: m, DriftCents: line.AmountCents - recomputed,
				Detail: fmt.Sprintf("meter %s line amount %d cents, but %.6f x %.10f recomputes to %d",
					m, line.AmountCents, line.Quantity, pinned, recomputed),
			})
		}
	}
	return findings, nil
}

// ratesEqual compares two per-unit rates with just enough slack for float64
// decimal parsing; catalog rates are exact decimal literals on both sides.
func ratesEqual(a, b float64) bool {
	if a == b {
		return true
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	return math.Abs(a-b) <= rateRelativeTolerance*scale
}
