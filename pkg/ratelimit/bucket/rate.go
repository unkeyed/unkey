package bucket

import (
	"math/bits"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// Rate allows `Limit` units per `Duration`, with a burst of `Limit` units.
//
// The burst tolerance is the full `Duration`: after an idle time, all `Limit`
// units fit at once, and a single request may cost the whole limit.
type Rate struct {
	Limit    int64
	Duration time.Duration
}

// Validate returns an error unless `Limit` and `Duration` are both positive.
func (r Rate) Validate() error {
	return assert.All(
		assert.Greater(r.Limit, 0, "rate limit must be positive"),
		assert.Greater(r.Duration, 0, "rate duration must be positive"),
	)
}

// costNs returns `⌈cost × Duration / Limit⌉`, the time that `cost` units use.
// It rounds up so that `Limit` requests never use less than `Duration`. The
// product uses 128 bits; the package documentation explains why plain integer
// math is not enough.
//
// The caller guarantees `0 ≤ cost ≤ Limit`, so the result is at most
// `Duration`.
func (r Rate) costNs(cost int64) int64 {
	productHigh, productLow := bits.Mul64(uint64(cost), uint64(r.Duration))
	quotient, remainder := bits.Div64(productHigh, productLow, uint64(r.Limit))
	// Rounding down would make every unit a little cheaper than its share,
	// and `Limit` requests could fit into less than `Duration`.
	if remainder > 0 {
		quotient++
	}
	return int64(quotient)
}

// remaining returns how many units fit at `nowNs` when the limit resets at
// `resetAtNs`. It is zero while `resetAtNs` is a full `Duration` or more
// ahead, which happens when pods over-admitted and the debt is not paid back
// yet.
func (r Rate) remaining(nowNs, resetAtNs int64) int64 {
	// The bucket holds `Duration` of time in total. The part between now and
	// `reset_at` is in use; the rest is free. A `reset_at` in the past means
	// the bucket is full, not that it holds more than `Duration`.
	aheadNs := max(resetAtNs-nowNs, 0)
	if aheadNs >= int64(r.Duration) {
		return 0
	}
	return r.unitsWithin(int64(r.Duration) - aheadNs)
}

// unitsWithin returns `⌊freeNs × Limit / Duration⌋`, the largest number of
// units whose cost fits into `freeNs`. Because [Rate.costNs] rounds up, `k`
// units fit exactly when `k ≤ freeNs × Limit / Duration`.
//
// The caller guarantees `0 ≤ freeNs ≤ Duration`, so the result is at most
// `Limit`.
func (r Rate) unitsWithin(freeNs int64) int64 {
	productHigh, productLow := bits.Mul64(uint64(freeNs), uint64(r.Limit))
	quotient, _ := bits.Div64(productHigh, productLow, uint64(r.Duration))
	return int64(quotient)
}
