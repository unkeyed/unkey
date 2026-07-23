package billingreconcile

import (
	"fmt"
	"math"
)

// overbillToleranceCents is the tolerance in the invoice-higher-than-CH
// direction, near-zero on purpose. Convergent last-aggregation pushes (the
// hourly push always sets the period's absolute month-to-date total, never
// increments it) make the invoice charging for more than ClickHouse ever
// recorded structurally impossible in normal operation, so this only absorbs
// the decimal-formatting noise between what was pushed (formatted to at most 12
// decimal places) and a fresh ClickHouse query now. One cent is generous
// headroom above that floor. Decision 7 withholds the dollar materiality bar
// from this direction: a systematic sub-cent overbill still surfaces in the
// monthly aggregate count, and a per-workspace sub-cent page would be noise.
const overbillToleranceCents = 1

// underbillFloorCents / underbillRelative set the CH-higher-than-invoice
// materiality bar: flag only past max($0.50, 0.5% of the meter's own invoice
// line). This is the expected, benign class (late ingest settling after the
// close's push), so it gets a real dollar floor.
const (
	underbillFloorCents = 50
	underbillRelative   = 0.005
)

// checkQuantities compares, per meter, the invoiced quantity against the
// re-derived ClickHouse quantity, direction-asymmetrically. A meter with no
// invoice line compares as invoiced quantity zero: under the convergent push a
// missing line means no usage was ever pushed, exactly the shortfall the
// late-data bar prices. Every delta is priced to integer cents with round()
// before comparison.
func checkQuantities(metered map[Meter]InvoiceLine, live map[Meter]float64) []Finding {
	var findings []Finding
	for _, m := range Meters() {
		line, hasLine := metered[m]
		var invoicedQty float64
		var lineAmountCents int64
		if hasLine {
			invoicedQty = line.Quantity
			lineAmountCents = line.AmountCents
		}
		rate := rateCents(m)

		deltaCents := int64(math.Round((invoicedQty - live[m]) * rate))
		switch {
		case deltaCents > overbillToleranceCents:
			findings = append(findings, Finding{
				Check:      CheckQuantity,
				Class:      VerdictOverbill,
				Meter:      m,
				DriftCents: deltaCents,
				Detail: fmt.Sprintf("meter %s: invoice bills %.6f but clickhouse records %.6f (+%d cents)",
					m, invoicedQty, live[m], deltaCents),
			})
		case -deltaCents > underbillThresholdCents(lineAmountCents):
			findings = append(findings, Finding{
				Check:      CheckQuantity,
				Class:      VerdictLateDataUnderbill,
				Meter:      m,
				DriftCents: deltaCents,
				Detail: fmt.Sprintf("meter %s: clickhouse records %.6f but invoice bills %.6f (%d cents, bar %d)",
					m, live[m], invoicedQty, deltaCents, underbillThresholdCents(lineAmountCents)),
			})
		}
	}
	return findings
}

// underbillThresholdCents is max($0.50, round(0.5% of the line)).
func underbillThresholdCents(lineAmountCents int64) int64 {
	pct := int64(math.Round(float64(lineAmountCents) * underbillRelative))
	if pct > underbillFloorCents {
		return pct
	}
	return underbillFloorCents
}
