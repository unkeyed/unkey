package deployspendcheck

// thresholds are the budget fractions, as percentages, that trigger an alert,
// ascending. Vercel's model: one budget, alerts at fixed percentages of it.
var thresholds = []int32{50, 75, 100}

// crossedThreshold returns the highest alert threshold (0, 50, 75 or 100) the
// overage has reached against the budget. 0 means no threshold reached yet.
// budget is assumed positive (the query filters out null budgets; the caller
// skips non-positive ones). overage is in fractional cents, so a partial cent
// past a threshold still counts.
func crossedThreshold(overageCents float64, budgetCents int64) int32 {
	pct := overageCents / float64(budgetCents) * 100
	var highest int32
	for _, t := range thresholds {
		if pct >= float64(t) {
			highest = t
		}
	}
	return highest
}
