package ratelimit

import (
	"math"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// Rate allows `Limit` units per `Duration`. The burst is the full `Limit`,
// and `Duration` is the time constant of the decay: a level falls by the
// factor e every `Duration`. Its methods only read the value, so they are
// safe to call concurrently.
type Rate struct {
	Limit    int64
	Duration time.Duration
}

const (
	// minDuration and maxDuration are the bounds of the API.
	minDuration = time.Second
	maxDuration = 30 * 24 * time.Hour

	// periodDurations is the length of a reference period in durations,
	// see `periodLengthMs`.
	periodDurations = 10
)

// Validate returns an error unless `Limit` is positive and `Duration` is a
// whole number of milliseconds between `minDuration` and `maxDuration`. The
// limiter keeps time in milliseconds, so every pod must compute the same
// period edges from the duration.
func (r Rate) Validate() error {
	return assert.All(
		assert.Greater(r.Limit, 0, "rate limit must be positive"),
		assert.GreaterOrEqual(r.Duration, minDuration, "rate duration must be at least one second"),
		assert.LessOrEqual(r.Duration, maxDuration, "rate duration must be at most 30 days"),
		assert.Equal(r.Duration%time.Millisecond, 0, "rate duration must be a whole number of milliseconds"),
	)
}

// periodLengthMs returns the length of the reference periods of a limit with
// `duration`: `periodDurations × duration`. Periods start at multiples of
// their length, so all pods share them. A request at `t` in the period that
// starts at `t0` adds `cost × e^((t − t0) / duration)` to its row, at most
// `cost × e^10 ≈ cost × 22,026`, so float64 keeps its 53 bits of precision
// far from overflow. The length is also how long a pod keeps writing one row,
// so a shorter period lets the rows of old pods fade sooner after a deploy.
// Every pod and version must compute the same length for a duration.
func periodLengthMs(duration time.Duration) int64 {
	return periodDurations * duration.Milliseconds()
}

// periodStartMs returns the start of the period that holds `nowMs`. It is
// safe to call concurrently.
func periodStartMs(nowMs, periodMs int64) int64 {
	return nowMs - nowMs%periodMs
}

// growthSince returns `e^((nowMs − periodStartMs) / duration)`, the factor
// that turns a level at `nowMs` into a scaled sum of the period that starts
// at `periodStartMs`, and back by dividing. It is safe to call concurrently.
func growthSince(nowMs, periodStartMs int64, duration time.Duration) float64 {
	return math.Exp(float64(nowMs-periodStartMs) / float64(duration.Milliseconds()))
}

// slackUnits is how far float rounding may move a level. Levels are sums of
// products of exponentials, each rounded to about 1e-16 of its value, so
// 1e-9 of the limit covers millions of terms. It stays below half a unit, so
// `Limit + 1` units never fit.
func (r Rate) slackUnits() float64 {
	return min(0.5, 1e-9*float64(r.Limit))
}

// effectiveLevel returns the level that decisions use: zero below one unit,
// the level otherwise. Without the cut, a level only reaches zero in the
// limit, and `remaining` would never return to `Limit`.
func effectiveLevel(level float64) float64 {
	if level < 1 {
		return 0
	}
	return level
}

// fits reports whether `units` fit on top of `level`.
func (r Rate) fits(level float64, units int64) bool {
	return effectiveLevel(level)+float64(units) <= float64(r.Limit)+r.slackUnits()
}

// remaining returns how many more units fit once `spent` units are added to
// `level`: `⌊Limit − effective level − spent⌋`, between 0 and `Limit`.
func (r Rate) remaining(level float64, spent int64) int64 {
	free := math.Floor(float64(r.Limit) - effectiveLevel(level) - float64(spent) + r.slackUnits())
	return int64(min(max(free, 0), float64(r.Limit)))
}

// nextUnitAfterMs returns how long after now one unit fits on top of a
// `level` that decays from now on. A level `L` falls to `x` after
// `Duration × ln(L / x)`. One unit fits once the level is at most
// `Limit − 1`, and for a limit of 1 once it is below 1, where the effective
// level drops to zero.
func (r Rate) nextUnitAfterMs(level float64) int64 {
	if r.fits(level, 1) {
		return 0
	}
	target := float64(r.Limit - 1)
	extraMs := 0.0
	if r.Limit == 1 {
		target = 1
		extraMs = 1
	}
	waitMs := math.Ceil(float64(r.Duration.Milliseconds())*math.Log(level/target)) + extraMs
	return int64(min(max(waitMs, 0), float64(math.MaxInt64/2)))
}
