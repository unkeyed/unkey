package ratelimit

import (
	"math/bits"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// Rate allows `Limit` units per `Duration`. The burst is the full `Limit`.
// Its methods only read the value, so they are safe to call concurrently.
type Rate struct {
	Limit    int64
	Duration time.Duration
}

// minDuration is the shortest `Duration`, the same as the API's. A shorter
// limit would need buckets narrower than one second, see `bucketWidthNs`.
const minDuration = time.Second

// Validate returns an error unless `Limit` is positive and `Duration` is at
// least `minDuration`.
func (r Rate) Validate() error {
	return assert.All(
		assert.Greater(r.Limit, 0, "rate limit must be positive"),
		assert.GreaterOrEqual(r.Duration, minDuration, "rate duration must be at least one second"),
	)
}

// bucketWidthNs returns the width of the buckets of the shared time grid:
// `max(1s, Duration / 1000)`. Every pod keeps about `(width + maxWriteDelay
// + maxClockSkew) / width` open rows per busy limit, so a width below one
// second multiplies the rows without making decisions more exact: pods see
// each other's units one to five seconds late anyway. Rows of two widths
// share no edges, so every pod and version must compute the same width for a
// duration. Do not change this function without storing the width in the
// rows.
func (r Rate) bucketWidthNs() int64 {
	return max(time.Second.Nanoseconds(), r.Duration.Nanoseconds()/1000)
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

// roundingSlackNs returns how far beyond `Duration` the `reset_at` of an
// allowed request may be ahead of now. `unitsToNs` charges each request less
// than 1 ns too much, so `Limit` requests at one instant are charged up to
// `Limit - 1` ns more than `Duration`. The slack stays below the cost of one
// unit, so `Limit + 1` units at one instant never fit.
func (r Rate) roundingSlackNs() int64 {
	return min(r.Limit, r.unitsToNs(1)) - 1
}

// remaining returns the most units that one request may take at `nowNs`.
// It is zero while `resetAtNs` is `Duration + roundingSlackNs` or more ahead,
// after an over-admission.
func (r Rate) remaining(nowNs, resetAtNs int64) int64 {
	aheadNs := max(resetAtNs-nowNs, 0)
	slackNs := r.roundingSlackNs()
	if aheadNs <= slackNs {
		return r.Limit
	}
	if aheadNs-slackNs >= r.Duration.Nanoseconds() { //faultline:equivalent at equality `unitsWithin(0)` is also 0
		return 0
	}
	return r.unitsWithin(r.Duration.Nanoseconds() - (aheadNs - slackNs))
}

// unitsWithin returns `⌊freeNs × Limit / Duration⌋`, the most units whose
// time fits into `freeNs`. The caller guarantees `0 ≤ freeNs ≤ Duration`.
func (r Rate) unitsWithin(freeNs int64) int64 {
	productHigh, productLow := bits.Mul64(uint64(freeNs), uint64(r.Limit))
	quotient, _ := bits.Div64(productHigh, productLow, uint64(r.Duration.Nanoseconds()))
	return int64(quotient)
}
