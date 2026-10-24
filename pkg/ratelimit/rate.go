package ratelimit

import (
	"math/bits"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// Rate allows `Limit` units per `Duration`. The burst is the full `Limit`.
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

// unitsToNs returns `⌈units × Duration / Limit⌉`, rounded up so that `Limit`
// requests never fit into less than `Duration`. The caller guarantees
// `0 ≤ units ≤ Limit`. See the package documentation for the 128-bit math.
func (r Rate) unitsToNs(units int64) int64 {
	productHigh, productLow := bits.Mul64(uint64(units), uint64(r.Duration.Nanoseconds()))
	quotient, remainder := bits.Div64(productHigh, productLow, uint64(r.Limit))
	if remainder > 0 {
		quotient++
	}
	return int64(quotient)
}

// remaining returns how many units fit at `nowNs`. It is zero while
// `resetAtNs` is a full `Duration` or more ahead, after an over-admission.
func (r Rate) remaining(nowNs, resetAtNs int64) int64 {
	aheadNs := max(resetAtNs-nowNs, 0)
	if aheadNs >= r.Duration.Nanoseconds() { //faultline:equivalent at equality `unitsWithin(0)` is also 0
		return 0
	}
	return r.unitsWithin(r.Duration.Nanoseconds() - aheadNs)
}

// unitsWithin returns `⌊freeNs × Limit / Duration⌋`, the most units whose
// time fits into `freeNs`. The caller guarantees `0 ≤ freeNs ≤ Duration`.
func (r Rate) unitsWithin(freeNs int64) int64 {
	productHigh, productLow := bits.Mul64(uint64(freeNs), uint64(r.Limit))
	quotient, _ := bits.Div64(productHigh, productLow, uint64(r.Duration.Nanoseconds()))
	return int64(quotient)
}
