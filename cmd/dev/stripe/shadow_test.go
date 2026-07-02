package stripe

import "testing"

func TestWithinTolerance(t *testing.T) {
	cases := []struct {
		name string
		a, b float64
		want bool
	}{
		{"both zero", 0, 0, true},
		{"zero vs nonzero drifts", 0, 1, false},
		{"identical", 3600, 3600, true},
		{"within 0.1% (push lag)", 100000, 100050, true},
		{"beyond 0.1% drifts", 100000, 100200, false},
		{"tiny absolute, equal", 0.234375, 0.234375, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := withinTolerance(tc.a, tc.b); got != tc.want {
				t.Fatalf("withinTolerance(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
