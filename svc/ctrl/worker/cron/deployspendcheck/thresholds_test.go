package deployspendcheck

import "testing"

func TestThresholdLevel(t *testing.T) {
	const budget = 10_000 // $100 budget in cents

	cases := []struct {
		name    string
		overage float64
		want    int32
	}{
		{"zero overage", 0, 0},
		{"just under 50%", 4_999, 0},
		{"exactly 50%", 5_000, 50},
		{"between 50 and 75", 6_000, 50},
		{"exactly 75%", 7_500, 75},
		{"just under 100%", 9_999, 75},
		{"exactly 100%", 10_000, 100},
		{"over 100%", 25_000, 100},
		{"fractional cent past 50%", 5_000.5, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := crossedThreshold(tc.overage, budget); got != tc.want {
				t.Fatalf("crossedThreshold(%v, %d) = %d, want %d", tc.overage, budget, got, tc.want)
			}
		})
	}
}
