package deployspendcheck

import (
	"testing"

	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

func TestThresholdLevel(t *testing.T) {
	// $100 budget, in micro-cents.
	const budget = 10_000 * deploybilling.MicroCentsPerCent

	cases := []struct {
		name    string
		overage int64
		want    int32
	}{
		{"zero overage", 0, 0},
		{"just under 50%", 4_999 * deploybilling.MicroCentsPerCent, 0},
		{"exactly 50%", 5_000 * deploybilling.MicroCentsPerCent, 50},
		{"between 50 and 75", 6_000 * deploybilling.MicroCentsPerCent, 50},
		{"exactly 75%", 7_500 * deploybilling.MicroCentsPerCent, 75},
		{"just under 100%", 9_999 * deploybilling.MicroCentsPerCent, 75},
		{"exactly 100%", 10_000 * deploybilling.MicroCentsPerCent, 100},
		{"over 100%", 25_000 * deploybilling.MicroCentsPerCent, 100},
		{"one micro-cent past 50%", 5_000*deploybilling.MicroCentsPerCent + 1, 50},
		{"one micro-cent under 50%", 5_000*deploybilling.MicroCentsPerCent - 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := crossedThreshold(tc.overage, budget); got != tc.want {
				t.Fatalf("crossedThreshold(%v, %d) = %d, want %d", tc.overage, budget, got, tc.want)
			}
		})
	}
}

// TestThresholdDegenerateBudget covers the degenerate budgets: a non-positive budget
// defines no threshold, and huge budgets must not overflow the cross-multiply
// into a false 100%.
func TestThresholdDegenerateBudget(t *testing.T) {
	t.Run("zero budget reports nothing crossed even at spend", func(t *testing.T) {
		// The pre-fix code returned 100 here (spend*100 >= 0 is always true),
		// which dispatched the workspace every tick forever as a no-op.
		if got := crossedThreshold(0, 0); got != 0 {
			t.Fatalf("crossedThreshold(0, 0) = %d, want 0", got)
		}
		if got := crossedThreshold(5_000*deploybilling.MicroCentsPerCent, 0); got != 0 {
			t.Fatalf("crossedThreshold(spend, 0) = %d, want 0", got)
		}
	})

	t.Run("negative operands report nothing crossed", func(t *testing.T) {
		if got := crossedThreshold(-1, 10_000*deploybilling.MicroCentsPerCent); got != 0 {
			t.Fatalf("crossedThreshold(-1, budget) = %d, want 0", got)
		}
	})

	// Budget past the int64 overflow bound for the cross-multiply. At micro-cent
	// scale threshold*budget and spend*100 overflow int64 once budgetMicroCents
	// exceeds math.MaxInt64/100 (roughly $922M/month). The pre-fix int64 math
	// wrapped there, so the comparison read as crossed at any spend, including
	// zero; the 128-bit math stays exact.
	t.Run("huge budget does not overflow into false 100", func(t *testing.T) {
		// 2e17 micro-cents == $2B/month, comfortably past the overflow bound and
		// even, so 50% is exact. 100*budget == 2e19 overflows int64 (max 9.22e18).
		const overflowBudget int64 = 200_000_000_000_000_000

		// Zero spend against an enormous budget: nothing crossed. This is the
		// exact regression: the pre-fix code returned 100 here.
		if got := crossedThreshold(0, overflowBudget); got != 0 {
			t.Fatalf("crossedThreshold(0, huge) = %d, want 0", got)
		}
		// A tiny spend against the enormous budget is still far below 50%.
		if got := crossedThreshold(1_000_000, overflowBudget); got != 0 {
			t.Fatalf("crossedThreshold(tiny, huge) = %d, want 0", got)
		}
		// Exactly 50% of the enormous budget must read as 50 and no higher: this
		// exercises the 128-bit compare at the scale the old code overflowed.
		if got := crossedThreshold(overflowBudget/2, overflowBudget); got != 50 {
			t.Fatalf("crossedThreshold(half, huge) = %d, want 50", got)
		}
		// One micro-cent short of 100% must still read 75, not 100.
		if got := crossedThreshold(overflowBudget-1, overflowBudget); got != 75 {
			t.Fatalf("crossedThreshold(just under full, huge) = %d, want 75", got)
		}
	})
}
