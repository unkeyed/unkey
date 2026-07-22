package deployspendcheck

import "math/bits"

// thresholds are the budget fractions, as percentages, that trigger an alert,
// ascending. Vercel's model: one budget, alerts at fixed percentages of it.
var thresholds = []int32{50, 75, 100}

// crossedThreshold returns the highest alert threshold (0, 50, 75 or 100) the
// gross spend has reached against the budget. 0 means no threshold reached yet.
// A non-positive budget defines no threshold and returns 0 (the dashboard
// blocks it, a direct DB write does not). The comparison is the exact
// cross-multiplied spend*100 >= threshold*budget in 128 bits: at micro-cent
// scale the products overflow int64 past roughly $922M/month, and a silent
// overflow would flip the comparison into instant 100% at zero spend.
func crossedThreshold(spendMicroCents, budgetMicroCents int64) int32 {
	if budgetMicroCents <= 0 || spendMicroCents < 0 {
		return 0
	}
	spendHi, spendLo := bits.Mul64(uint64(spendMicroCents), 100)
	var highest int32
	for _, t := range thresholds {
		budgetHi, budgetLo := bits.Mul64(uint64(budgetMicroCents), uint64(t))
		// 128-bit spend*100 >= threshold*budget.
		if spendHi > budgetHi || (spendHi == budgetHi && spendLo >= budgetLo) {
			highest = t
		}
	}
	return highest
}
