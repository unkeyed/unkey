package ratelimit

import (
	"math"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// rowKey identifies one row of a fingerprint. Reads sum the rows of all
// other pods into one row per period with an empty pod ID.
type rowKey struct {
	periodStartNs int64
	podID         string
}

// row is one row of `ratelimit_state` as this pod knows it.
type row struct {
	// scaledSum is `Σ cost × e^((t − period start) / Duration)` over the
	// admitted requests of the row. It only grows.
	scaledSum float64

	// own is true if this pod wrote the row, so this pod writes it.
	own bool

	// committedSum is the scaled sum that the store acknowledged. It is only
	// used for own rows.
	committedSum float64
}

// uncommitted reports whether the store has not acknowledged all of an own
// row. It is safe to call concurrently.
func (r row) uncommitted() bool {
	return r.own && r.scaledSum > r.committedSum
}

// levelAt returns the level of the row at `nowNs`. It is safe to call
// concurrently.
func (r row) levelAt(key rowKey, nowNs int64, duration time.Duration) float64 {
	return r.scaledSum / growthSince(nowNs, key.periodStartNs, duration)
}

// levelSum is the level of a set of rows, kept as one scaled sum against
// the start of the current period, so a decision needs one exponential:
//
//	level(now) = sum / e^((now − referenceNs) / Duration)
type levelSum struct {
	referenceNs int64
	sum         float64
}

// level returns the level at `nowNs`. It is safe to call concurrently.
func (s levelSum) level(nowNs int64, duration time.Duration) float64 {
	return s.sum / growthSince(nowNs, s.referenceNs, duration)
}

// computeLevelSum adds up the rows against `referenceNs`. A row's scaled
// sum moves to another period start by the factor
// `e^((row start − reference) / Duration)`, which is exact, so the order of
// the rows and of the requests in them does not matter. It returns an error
// for a negative or non-finite scaled sum. It only reads its arguments, so it
// is safe to call concurrently.
func computeLevelSum(rows map[rowKey]row, referenceNs int64, duration time.Duration) (levelSum, error) {
	result := levelSum{referenceNs: referenceNs, sum: 0}
	for key, r := range rows {
		err := assert.All(
			assert.GreaterOrEqual(r.scaledSum, 0, "scaled sum must not be negative"),
			assert.False(math.IsInf(r.scaledSum, 0) || math.IsNaN(r.scaledSum), "scaled sum must be finite"),
		)
		if err != nil {
			return levelSum{}, err
		}
		result.sum += r.scaledSum * growthSince(key.periodStartNs, referenceNs, duration)
	}
	return result, nil
}

// decision is the result of `decide`.
type decision struct {
	allowed bool

	// periodStartNs is the period of the request, and addedSum what it adds
	// to this pod's row of that period. addedSum is zero for a denied request
	// or zero units; then nothing is written.
	periodStartNs int64
	addedSum      float64

	// levelBefore is the level at the time of the request, and levelAfter
	// the level once the decision is applied.
	levelBefore float64
	levelAfter  float64
}

// decide applies the rule to a request of `units` at `nowNs`:
//
//	allow if effective level + units ≤ Limit
//
// `current` must be referenced to the period of `nowNs`. It is pure, so a
// caller can decide several rate limits before applying any. It returns an
// error for invalid input. It is safe to call concurrently.
func decide(current levelSum, nowNs int64, rate Rate, units int64) (decision, error) {
	err := assert.All(
		rate.Validate(),
		assert.GreaterOrEqual(units, 0, "units must not be negative"),
		assert.Greater(nowNs, 0, "time must be after the Unix epoch"),
		assert.Equal(current.referenceNs, periodStartNs(nowNs, periodLengthNs(rate.Duration)), "the level must be referenced to the current period"),
	)
	if err != nil {
		return decision{}, err
	}
	growth := growthSince(nowNs, current.referenceNs, rate.Duration)
	level := current.sum / growth
	result := decision{
		// Above 2^53, float64 cannot tell `Limit + 1` from `Limit`.
		allowed:       units <= rate.Limit && rate.fits(level, units),
		periodStartNs: current.referenceNs,
		addedSum:      0,
		levelBefore:   level,
		levelAfter:    level,
	}
	if result.allowed && units > 0 {
		result.addedSum = float64(units) * growth
		result.levelAfter = level + float64(units)
	}
	return result, nil
}
