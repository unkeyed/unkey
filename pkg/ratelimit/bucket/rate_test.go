package bucket

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestRateCostIsExactForExtremeRates guarantees that the full limit always
// uses exactly one Duration, and that a single unit is never rounded down to
// zero, for very high and very low rates.
func TestRateCostIsExactForExtremeRates(t *testing.T) {
	week := 7 * 24 * time.Hour
	cases := []struct {
		name   string
		rate   Rate
		cost   int64
		wantNs int64
	}{
		{"one of 162M per minute rounds up", Rate{Limit: 162_000_000, Duration: time.Minute}, 1, 371},
		{"all of 162M per minute is one minute", Rate{Limit: 162_000_000, Duration: time.Minute}, 162_000_000, int64(time.Minute)},
		{"one per week is one week", Rate{Limit: 1, Duration: week}, 1, int64(week)},
		{"max limit over max duration", Rate{Limit: math.MaxInt64, Duration: math.MaxInt64}, math.MaxInt64, math.MaxInt64},
		{"cost zero uses no time", Rate{Limit: 10, Duration: time.Second}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantNs, tc.rate.costNs(tc.cost))
		})
	}
}

// FuzzRateMatchesExactArithmetic guarantees that the 128-bit math of costNs
// equals ⌈cost × Duration / Limit⌉ for every valid rate, and that unitsWithin
// is the largest number of units whose cost fits. A mistake here lets the
// limiter admit more than the limit or report a wrong remaining count.
func FuzzRateMatchesExactArithmetic(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		rate := Rate{
			Limit:    int64(c.Uint64()>>1) + 1,
			Duration: time.Duration(c.Uint64()>>1) + 1,
		}
		if rate.Limit < 0 || rate.Duration < 0 {
			t.Skip("wrapped around")
		}
		cost := int64(c.Uint64() % (uint64(rate.Limit) + 1))
		freeNs := int64(c.Uint64() % (uint64(rate.Duration) + 1))

		exact := new(big.Int).Mul(big.NewInt(cost), big.NewInt(int64(rate.Duration)))
		exact.Add(exact, big.NewInt(rate.Limit-1))
		exact.Quo(exact, big.NewInt(rate.Limit))
		require.Equal(t, exact.Int64(), rate.costNs(cost))

		units := rate.unitsWithin(freeNs)
		require.LessOrEqual(t, units, rate.Limit)
		require.LessOrEqual(t, rate.costNs(units), freeNs)
		if units < rate.Limit {
			require.Greater(t, rate.costNs(units+1), freeNs)
		}
	})
}
