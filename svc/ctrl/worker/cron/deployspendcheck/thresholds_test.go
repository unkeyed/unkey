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
