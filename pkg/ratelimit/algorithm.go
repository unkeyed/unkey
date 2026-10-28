package ratelimit

import (
	"cmp"
	"math"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// rowKey identifies one row of a fingerprint. The base row has bucket start
// 0 and an empty pod ID.
type rowKey struct {
	bucketStartNs int64
	podID         string
}

// baseRowKey is the key of the row that cleanup merges closed buckets into.
var baseRowKey = rowKey{bucketStartNs: 0, podID: ""}

// row is one row of `ratelimit_state` as this pod knows it.
type row struct {
	// consumedNs only grows. A row with zero consumed time counts as no row:
	// MySQL stores it when it refuses the insert of a closed bucket.
	consumedNs int64

	// resetAtNs is the GCRA `reset_at` of the row's requests alone:
	//
	//	max over requests i of the row of
	//	    (time of i + consumed time of the row's requests at or after i)
	//
	// For the base row it is the `reset_at` of all merged buckets.
	resetAtNs int64

	// own is true if this pod wrote the row, so this pod writes it.
	own bool

	// committedNs is the consumed time that the store acknowledged, and
	// committedResetAtNs the reset time written with it. They are only used
	// for own rows.
	committedNs        int64
	committedResetAtNs int64
}

// unacknowledgedNs returns the consumed time of an own row that the store
// has not acknowledged, and zero for a peer row. It is safe to call
// concurrently.
func (r row) unacknowledgedNs() int64 {
	if !r.own {
		return 0
	}
	return r.consumedNs - r.committedNs
}

// bucketTotals sums the rows of one bucket.
type bucketTotals struct {
	startNs    int64
	consumedNs int64

	// ownResetAtNs is the `resetAtNs` of this pod's row, or zero without one.
	ownResetAtNs int64

	// maxPeerResetAtNs is the largest `resetAtNs` of the other rows.
	maxPeerResetAtNs int64
}

// resetAtNs returns the `reset_at` of the bucket's requests alone, or the
// largest value that the rows prove:
//
//	max(start + consumed, own reset_at, largest peer reset_at)
//
// Every request of the bucket is at or after its start, and each row's
// `reset_at` counts only part of the bucket's requests, so the result is
// never later than the GCRA `reset_at` of the bucket's requests. It is
// earlier by less than the bucket width, and exact when one pod wrote the
// bucket. It is safe to call concurrently.
func (b bucketTotals) resetAtNs() (int64, error) {
	startPlusConsumedNs, err := checkedAdd(b.startNs, b.consumedNs)
	if err != nil {
		return 0, err
	}
	return max(startPlusConsumedNs, b.ownResetAtNs, b.maxPeerResetAtNs), nil
}

// resetTimes holds `reset_at` and the values that `decide` needs to add a
// request to the current bucket in O(1).
//
// The candidate of a bucket is its `resetAtNs` plus the consumed time of all
// newer buckets, and `reset_at` is the largest candidate. A request in the
// current bucket raises the candidates of all older buckets by its consumed
// time, changes the candidate of the current bucket, and leaves the newer
// buckets alone. Newer buckets exist when a peer's clock is ahead.
type resetTimes struct {
	resetAtNs int64

	// current holds the bucket that the next request goes into.
	current bucketTotals

	// maxCandidateOlderNs is the largest candidate of the buckets older than
	// `current`, or zero without one.
	maxCandidateOlderNs int64

	// maxCandidateNewerNs is the largest candidate of the buckets newer than
	// `current`, or zero without one.
	maxCandidateNewerNs int64

	// consumedNewerNs is the consumed time of the buckets newer than
	// `current`.
	consumedNewerNs int64

	// newestBucketStartNs is the start of the newest bucket with consumed
	// time, or zero without one.
	newestBucketStartNs int64
}

// decision is the result of `decide`.
type decision struct {
	allowed bool

	// consumedNs is zero for a denied request or zero `units`; then nothing is
	// written.
	consumedNs int64

	resetAtNs int64

	// remaining is how many units one more request could use right now.
	remaining int64

	// bucket is the current bucket after the request.
	bucket bucketTotals
}

// computeResetTimes computes the reset times of all rows in O(n log n), with
// `currentBucketStartNs` as the bucket of the next request:
//
//	reset_at = max over buckets b of
//	           (resetAt of b + consumed time of buckets newer than b)
//
// For example, with buckets of 10 s:
//
//	bucket     rows (consumed, reset_at)    resetAt of bucket   candidate
//	12:00:20   own (3s, 12:00:24)           12:00:24            12:00:24
//	12:00:10   A (4s, 12:00:19), B (2s, …)  12:00:19            12:00:22
//	base       (20s, 11:59:55)              11:59:55            12:00:04
//
// It returns an error for a negative value or an int64 overflow. It only
// reads its arguments, so it is safe to call concurrently.
func computeResetTimes(rows map[rowKey]row, currentBucketStartNs int64) (resetTimes, error) {
	buckets := make(map[int64]*bucketTotals)
	for key, r := range rows {
		err := assert.All(
			assert.GreaterOrEqual(key.bucketStartNs, 0, "bucket start must not be negative"),
			assert.GreaterOrEqual(r.consumedNs, 0, "consumed time must not be negative"),
			assert.GreaterOrEqual(r.resetAtNs, 0, "reset time must not be negative"),
		)
		if err != nil {
			return resetTimes{}, err
		}
		if r.consumedNs == 0 {
			continue
		}
		b, ok := buckets[key.bucketStartNs]
		if !ok {
			b = &bucketTotals{startNs: key.bucketStartNs, consumedNs: 0, ownResetAtNs: 0, maxPeerResetAtNs: 0}
			buckets[key.bucketStartNs] = b
		}
		b.consumedNs, err = checkedAdd(b.consumedNs, r.consumedNs)
		if err != nil {
			return resetTimes{}, err
		}
		if r.own {
			b.ownResetAtNs = r.resetAtNs
		} else {
			b.maxPeerResetAtNs = max(b.maxPeerResetAtNs, r.resetAtNs)
		}
	}
	newestFirst := slices.SortedFunc(func(yield func(*bucketTotals) bool) {
		for _, b := range buckets {
			if !yield(b) {
				return
			}
		}
	}, func(left, right *bucketTotals) int {
		return cmp.Compare(right.startNs, left.startNs)
	})

	result := resetTimes{
		resetAtNs:           0,
		current:             bucketTotals{startNs: currentBucketStartNs, consumedNs: 0, ownResetAtNs: 0, maxPeerResetAtNs: 0},
		maxCandidateOlderNs: 0,
		maxCandidateNewerNs: 0,
		consumedNewerNs:     0,
		newestBucketStartNs: 0,
	}
	if len(newestFirst) > 0 {
		result.newestBucketStartNs = newestFirst[0].startNs
	}
	var consumedNewerNs int64
	for _, b := range newestFirst {
		bucketResetAtNs, err := b.resetAtNs()
		if err != nil {
			return resetTimes{}, err
		}
		candidateNs, err := checkedAdd(bucketResetAtNs, consumedNewerNs)
		if err != nil {
			return resetTimes{}, err
		}
		result.resetAtNs = max(result.resetAtNs, candidateNs)
		switch {
		case b.startNs < currentBucketStartNs:
			result.maxCandidateOlderNs = max(result.maxCandidateOlderNs, candidateNs)
		case b.startNs > currentBucketStartNs:
			result.maxCandidateNewerNs = max(result.maxCandidateNewerNs, candidateNs)
			result.consumedNewerNs = consumedNewerNs + b.consumedNs
		default:
			result.current = *b
		}
		consumedNewerNs, err = checkedAdd(consumedNewerNs, b.consumedNs)
		if err != nil {
			return resetTimes{}, err
		}
	}
	return result, nil
}

// movedTo returns the reset times with `bucketStartNs` as the current bucket
// in O(1), or false if that needs `computeResetTimes`. It only reads its
// arguments, so it is safe to call concurrently.
func (current resetTimes) movedTo(bucketStartNs int64) (resetTimes, bool) {
	if bucketStartNs == current.current.startNs {
		return current, true
	}
	if bucketStartNs <= current.newestBucketStartNs {
		return current, false
	}
	// Every bucket with consumed time is older than the new current one.
	return resetTimes{
		resetAtNs:           current.resetAtNs,
		current:             bucketTotals{startNs: bucketStartNs, consumedNs: 0, ownResetAtNs: 0, maxPeerResetAtNs: 0},
		maxCandidateOlderNs: current.resetAtNs,
		maxCandidateNewerNs: 0,
		consumedNewerNs:     0,
		newestBucketStartNs: current.newestBucketStartNs,
	}, true
}

// decide applies the GCRA rule to a request of `units` at `nowNs`, which must
// be in the current bucket of `current`. It is pure, so a caller can decide
// several rate limits before applying any. It returns an error for invalid
// input or an int64 overflow. It is safe to call concurrently.
//
// The request is allowed if `max(reset_at, now) + consumed` stays at most
// `Duration` plus `roundingSlackNs` ahead of now. The new `reset_at` follows from the formula of
// `computeResetTimes` and is at most that sum: less when a peer's row in the
// current bucket may hold requests from before now, see
// `bucketTotals.resetAtNs`.
func decide(current resetTimes, nowNs int64, rate Rate, units int64) (decision, error) {
	err := assert.All(
		rate.Validate(),
		assert.GreaterOrEqual(units, 0, "units must not be negative"),
		assert.Greater(nowNs, 0, "time must be after the Unix epoch"),
		assert.Equal(current.current.startNs, bucketStartNs(nowNs, rate.bucketWidthNs()), "now must be in the current bucket"),
	)
	if err != nil {
		return decision{}, err
	}

	unchanged := decision{
		allowed:    false, //faultline:equivalent the zero value; exhaustruct requires the field
		consumedNs: 0,     //faultline:equivalent the zero value; exhaustruct requires the field
		resetAtNs:  current.resetAtNs,
		remaining:  rate.remaining(nowNs, current.resetAtNs),
		bucket:     current.current,
	}
	// More than `Limit` never fits. Checked first so `unitsToNs` cannot panic.
	if units > rate.Limit {
		return unchanged, nil
	}
	consumedNs := rate.unitsToNs(units)
	gcraResetAtNs, err := checkedAdd(max(current.resetAtNs, nowNs), consumedNs)
	if err != nil {
		return decision{}, err
	}
	if gcraResetAtNs-nowNs-rate.Duration.Nanoseconds() > rate.roundingSlackNs() {
		return unchanged, nil
	}
	// Zero units only ask whether the limit is open; they write nothing.
	if units == 0 {
		unchanged.allowed = true
		return unchanged, nil
	}

	// gcraResetAtNs bounds every sum below, so none overflows.
	bucket := current.current
	bucket.consumedNs += consumedNs
	bucket.ownResetAtNs = max(bucket.ownResetAtNs, nowNs) + consumedNs
	bucketResetAtNs, err := bucket.resetAtNs()
	if err != nil {
		return decision{}, err
	}
	resetAtNs := max(bucketResetAtNs+current.consumedNewerNs, current.maxCandidateNewerNs)
	if current.maxCandidateOlderNs > 0 {
		resetAtNs = max(resetAtNs, current.maxCandidateOlderNs+consumedNs)
	}
	return decision{
		allowed:    true,
		consumedNs: consumedNs,
		resetAtNs:  resetAtNs,
		remaining:  rate.remaining(nowNs, resetAtNs),
		bucket:     bucket,
	}, nil
}

// resetTimesAfter returns the reset times after an allowed decision with
// positive `consumedNs` was applied. The decision must come from `decide` on
// `current`. It only reads its arguments, so it is safe to call concurrently.
func resetTimesAfter(current resetTimes, d decision) resetTimes {
	next := current
	next.resetAtNs = d.resetAtNs
	next.current = d.bucket
	if next.maxCandidateOlderNs > 0 {
		next.maxCandidateOlderNs += d.consumedNs
	}
	next.newestBucketStartNs = max(next.newestBucketStartNs, d.bucket.startNs)
	return next
}

// bucketStartNs returns the start of the bucket that holds `nowNs`. Bucket
// edges are multiples of the width, so all pods share them. It is safe to
// call concurrently.
func bucketStartNs(nowNs, bucketWidthNs int64) int64 {
	return nowNs - nowNs%bucketWidthNs
}

// checkedAdd returns `a + b` for non-negative `a` and `b`, or an error on
// overflow. A wrapped `reset_at` would open the rate limit. It is safe to call
// concurrently.
func checkedAdd(a, b int64) (int64, error) {
	if a > math.MaxInt64-b {
		return 0, fault.New("sum overflows int64") //faultline:equivalent callers discard the value when there is an error
	}
	return a + b, nil
}
