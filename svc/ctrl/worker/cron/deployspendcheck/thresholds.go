package deployspendcheck

// thresholds are the budget fractions, as percentages, that trigger an alert,
// ascending. Vercel's model: one budget, alerts at fixed percentages of it.
var thresholds = []int32{50, 75, 100}

// crossedThreshold returns the highest alert threshold (0, 50, 75 or 100) the
// overage has reached against the budget. 0 means no threshold reached yet.
// budget is assumed positive (the query filters out null budgets; the caller
// skips non-positive ones). Both amounts are integer micro-cents and the
// comparison cross-multiplies, so it is exact: no division, no floats, and a
// single micro-cent past a threshold still counts.
func crossedThreshold(overageMicroCents, budgetMicroCents int64) int32 {
	var highest int32
	for _, t := range thresholds {
		if overageMicroCents*100 >= int64(t)*budgetMicroCents {
			highest = t
		}
	}
	return highest
}
