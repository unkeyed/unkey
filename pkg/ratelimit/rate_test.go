package ratelimit

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestRateUnitsToNsIsExactForExtremeRates guarantees that the full limit
// always uses exactly one `Duration`, and that a single unit is never rounded down to
// zero, for very high and very low rates.
func TestRateUnitsToNsIsExactForExtremeRates(t *testing.T) {
	cases := []struct {
		name   string
		rate   Rate
		units  int64
		wantNs int64
	}{
		{"one of 162M per minute rounds up", Rate{Limit: 162_000_000, Duration: time.Minute}, 1, 371},
		{"all of 162M per minute is one minute", Rate{Limit: 162_000_000, Duration: time.Minute}, 162_000_000, time.Minute.Nanoseconds()},
		{"one per week is one week", Rate{Limit: 1, Duration: 7 * 24 * time.Hour}, 1, (7 * 24 * time.Hour).Nanoseconds()},
		{"max limit over max duration", Rate{Limit: math.MaxInt64, Duration: math.MaxInt64}, math.MaxInt64, math.MaxInt64},
		{"zero units use no time", Rate{Limit: 10, Duration: time.Second}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantNs, tc.rate.unitsToNs(tc.units))
		})
	}
}

// FuzzRateMatchesExactArithmetic guarantees that the 128-bit math of
// `unitsToNs` equals `⌈units × Duration / Limit⌉` for every valid rate, and
// that `unitsWithin` is the largest number of units whose time fits. A mistake here lets the
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
		units := int64(c.Uint64() % (uint64(rate.Limit) + 1))
		freeNs := int64(c.Uint64() % (uint64(rate.Duration.Nanoseconds()) + 1))

		exact := new(big.Int).Mul(big.NewInt(units), big.NewInt(rate.Duration.Nanoseconds()))
		exact.Add(exact, big.NewInt(rate.Limit-1))
		exact.Quo(exact, big.NewInt(rate.Limit))
		require.Equal(t, exact.Int64(), rate.unitsToNs(units))

		fittingUnits := rate.unitsWithin(freeNs)
		require.LessOrEqual(t, fittingUnits, rate.Limit)
		require.LessOrEqual(t, rate.unitsToNs(fittingUnits), freeNs)
		if fittingUnits < rate.Limit {
			require.Greater(t, rate.unitsToNs(fittingUnits+1), freeNs)
		}
	})
}
