package ratelimit

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// emptyResetTimes returns the reset times of a limit without rows, with the
// bucket of `nowNs` as the current one.
func emptyResetTimes(nowNs int64, rate Rate) resetTimes {
	var empty resetTimes
	empty.current.startNs = bucketStartNs(nowNs, rate.bucketWidthNs())
	return empty
}

// TestComputeResetTimesReplaysItsExample guarantees that the example in the
// documentation of `computeResetTimes` is what the code does.
func TestComputeResetTimesReplaysItsExample(t *testing.T) {
	noonNs := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	at := func(seconds int64) int64 { return noonNs + seconds*time.Second.Nanoseconds() }
	secondsNs := func(seconds int64) int64 { return seconds * time.Second.Nanoseconds() }
	rows := map[rowKey]row{
		{bucketStartNs: at(20), podID: "own"}: {consumedNs: secondsNs(3), resetAtNs: at(24), own: true, committedNs: 0, committedResetAtNs: 0},
		{bucketStartNs: at(10), podID: "a"}:   {consumedNs: secondsNs(4), resetAtNs: at(19), own: false, committedNs: 0, committedResetAtNs: 0},
		{bucketStartNs: at(10), podID: "b"}:   {consumedNs: secondsNs(2), resetAtNs: at(15), own: false, committedNs: 0, committedResetAtNs: 0},
		baseRowKey:                            {consumedNs: secondsNs(20), resetAtNs: at(-5), own: false, committedNs: 0, committedResetAtNs: 0},
	}
	result, err := computeResetTimes(rows, at(10))
	require.NoError(t, err)
	require.Equal(t, resetTimes{
		resetAtNs:           at(24),
		current:             bucketTotals{startNs: at(10), consumedNs: secondsNs(6), ownResetAtNs: 0, maxPeerResetAtNs: at(19)},
		maxCandidateOlderNs: at(4),
		maxCandidateNewerNs: at(24),
		consumedNewerNs:     secondsNs(3),
		newestBucketStartNs: at(20),
	}, result)
	require.Equal(t, at(24), resetAtByFormula(rows))
}

// TestComputeResetTimesRejectsInvalidRows guarantees that corrupt rows return
// an error instead of a wrong `reset_at`. A sum beyond int64 would wrap into
// the past and open the rate limit.
func TestComputeResetTimesRejectsInvalidRows(t *testing.T) {
	cases := []struct {
		name string
		rows map[rowKey]row
	}{
		{"negative bucket start", map[rowKey]row{{bucketStartNs: -1, podID: "a"}: {consumedNs: 1}}},
		{"negative consumed time", map[rowKey]row{{bucketStartNs: 1, podID: "a"}: {consumedNs: -1}}},
		{"negative reset time", map[rowKey]row{{bucketStartNs: 1, podID: "a"}: {consumedNs: 1, resetAtNs: -1}}},
		{"consumed sum beyond int64", map[rowKey]row{
			{bucketStartNs: 1, podID: "a"}: {consumedNs: math.MaxInt64 - 1},
			{bucketStartNs: 1, podID: "b"}: {consumedNs: 2},
		}},
		{"bucket start plus consumed beyond int64", map[rowKey]row{{bucketStartNs: math.MaxInt64 - 5, podID: "a"}: {consumedNs: 6}}},
		{"reset time plus newer consumed beyond int64", map[rowKey]row{
			{bucketStartNs: 1, podID: "a"}: {consumedNs: 1, resetAtNs: math.MaxInt64 - 1},
			{bucketStartNs: 2, podID: "a"}: {consumedNs: 2},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := computeResetTimes(tc.rows, 1)
			require.Error(t, err)
		})
	}
}

// TestDecideIsExactForLargeRates guarantees that `decide` itself uses the
// exact 128-bit math for the rates in the package documentation, whose
// `Limit × Duration` does not fit into 64 bits: the full limit uses exactly
// one `Duration`, and an idle rate limit reports the full limit as remaining.
func TestDecideIsExactForLargeRates(t *testing.T) {
	rates := []Rate{
		{Limit: 10_000, Duration: 30 * 24 * time.Hour},
		{Limit: 400_000_000, Duration: time.Minute},
	}
	for _, rate := range rates {
		t.Run(fmt.Sprintf("%d per %s", rate.Limit, rate.Duration), func(t *testing.T) {
			idle, err := decide(emptyResetTimes(1, rate), 1, rate, 0)
			require.NoError(t, err)
			require.Equal(t, rate.Limit, idle.remaining)

			full, err := decide(emptyResetTimes(1, rate), 1, rate, rate.Limit)
			require.NoError(t, err)
			require.True(t, full.allowed)
			require.Equal(t, rate.Duration.Nanoseconds(), full.consumedNs)
			require.Zero(t, full.remaining)
		})
	}
}

// TestDecideAtTheEdges guarantees that `decide` rejects invalid input,
// decides exactly up to the int64 limit, and returns an error instead of
// wrapping around one nanosecond later. It also guarantees that a huge
// `units` is denied before the 128-bit division, which would panic.
func TestDecideAtTheEdges(t *testing.T) {
	cases := []struct {
		name        string
		nowNs       int64
		rate        Rate
		units       int64
		wantErr     bool
		wantAllowed bool
	}{
		{"zero time", 0, Rate{Limit: 1, Duration: time.Second}, 1, true, false},
		{"negative units", 1, Rate{Limit: 1, Duration: time.Second}, -1, true, false},
		{"zero limit", 1, Rate{Limit: 0, Duration: time.Second}, 1, true, false},
		{"negative limit", 1, Rate{Limit: -1, Duration: time.Second}, 1, true, false},
		{"duration below one second", 1, Rate{Limit: 1, Duration: time.Second - 1}, 1, true, false},
		{"reset_at exactly at the int64 maximum", math.MaxInt64 - time.Second.Nanoseconds(), Rate{Limit: 1, Duration: time.Second}, 1, false, true},
		{"reset_at one past the int64 maximum", math.MaxInt64 - time.Second.Nanoseconds() + 1, Rate{Limit: 1, Duration: time.Second}, 1, true, false},
		{"huge units on a huge duration", 1, Rate{Limit: 1, Duration: math.MaxInt64}, math.MaxInt64, false, false},
		{"smallest valid rate", 1, Rate{Limit: 1, Duration: time.Second}, 1, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := decide(emptyResetTimes(tc.nowNs, tc.rate), tc.nowNs, tc.rate, tc.units)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAllowed, result.allowed)
		})
	}
}

// TestDecideRejectsAnotherBucket guarantees that `decide` never adds a
// request to a bucket that does not hold `now`.
func TestDecideRejectsAnotherBucket(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	nowNs := int64(time.Hour)
	_, err := decide(emptyResetTimes(nowNs+time.Second.Nanoseconds(), rate), nowNs, rate, 1)
	require.Error(t, err)
}

// TestDecideReplaysTheDocumentationTables guarantees that the worked
// examples in the package documentation are what the code does: the
// table for 10 per 10 seconds on one pod, and two pods that over-admitted
// and pay it back.
func TestDecideReplaysTheDocumentationTables(t *testing.T) {
	noonNs := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	secondNs := time.Second.Nanoseconds()
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	type step struct {
		atSeconds        int64
		units            int64
		wantAllowed      bool
		wantResetAtAfter int64
	}
	replay := func(t *testing.T, e *entry, steps []step) {
		t.Helper()
		for index, s := range steps {
			result, err := e.check(noonNs+s.atSeconds*secondNs, rate, s.units)
			require.NoError(t, err)
			require.Equal(t, s.wantAllowed, result.allowed, "step %d", index)
			e.apply(result, "own")
			require.Equal(t, noonNs+s.wantResetAtAfter*secondNs, e.resetTimes.resetAtNs, "step %d", index)
		}
	}

	t.Run("one pod", func(t *testing.T) {
		e := newEntry(rate.Duration)
		replay(t, e, []step{
			{0, 1, true, 1},
			{0, 9, true, 10},
			{0, 1, false, 10},
			{1, 1, true, 11},
			{30, 1, true, 31},
		})
	})

	t.Run("over-admission is paid back", func(t *testing.T) {
		e := newEntry(rate.Duration)
		replay(t, e, []step{{0, 10, true, 10}})
		require.NoError(t, e.replaceRows([]StoredRow{{
			BucketStartNs:  noonNs,
			PodID:          "own",
			ConsumedNs:     10 * secondNs,
			ResetAtNs:      noonNs + 10*secondNs,
			CompactAfterNs: compactAfterNs(noonNs, secondNs),
		}, {
			BucketStartNs:  noonNs,
			PodID:          "pod_b",
			ConsumedNs:     10 * secondNs,
			ResetAtNs:      noonNs + 10*secondNs,
			CompactAfterNs: compactAfterNs(noonNs, secondNs),
		}}, "own", noonNs))
		require.Equal(t, noonNs+20*secondNs, e.resetTimes.resetAtNs)
		replay(t, e, []step{
			{10, 1, false, 20},
			{10, 0, true, 20},
			{11, 1, true, 21},
		})
	})
}

// FuzzDecideFollowsGCRARule guarantees the core property of the limiter
// math. For any history of peer rows, stale reads and requests:
//   - a request is allowed exactly if `max(reset_at, now) + consumed` stays
//     within one `Duration` of now, with `reset_at` by the formula over all
//     rows;
//   - an allowed request moves `reset_at` to the formula over the rows after
//     it, which is never later than `max(reset_at, now) + consumed` and
//     never earlier than before;
//   - the cached reset times always equal `computeResetTimes`, so the O(1)
//     updates in `decide`, `movedTo` and `resetTimesAfter` never drift;
//   - a denied request or a request of zero `units` changes nothing.
func FuzzDecideFollowsGCRARule(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		stepNs := (10 * time.Millisecond).Nanoseconds()
		rate := Rate{
			Limit:    int64(c.Uint8()%20) + 1,
			Duration: time.Second + time.Duration(c.Uint8()%200)*50*time.Millisecond,
		}
		widthNs := rate.bucketWidthNs()
		e := newEntry(rate.Duration)
		nowNs := time.Hour.Nanoseconds()
		peers := map[rowKey]StoredRow{}

		for range 100 {
			switch c.Uint8() % 5 {
			case 0:
				nowNs += int64(c.Uint8()) * stepNs
			case 1:
				// A peer row in a bucket near now, also ahead of it as for a
				// peer whose clock is ahead.
				key := rowKey{
					bucketStartNs: bucketStartNs(max(1, nowNs+int64(c.Int8())*widthNs/4), widthNs),
					podID:         fmt.Sprintf("peer_%d", c.Uint8()%3),
				}
				stored := peers[key]
				consumedNs := int64(c.Uint8()) * stepNs
				stored.BucketStartNs = key.bucketStartNs
				stored.PodID = key.podID
				stored.ConsumedNs += consumedNs
				stored.ResetAtNs = max(stored.ResetAtNs, key.bucketStartNs+int64(c.Uint8())%widthNs) + consumedNs
				peers[key] = stored
				require.NoError(t, e.replaceRows(storedValues(peers), "own", nowNs))
			case 2:
				// A read that misses some peer rows, as from a stale replica
				// or after cleanup deleted them.
				var read []StoredRow
				for _, stored := range storedValues(peers) {
					if c.Bool() {
						read = append(read, stored)
					}
				}
				require.NoError(t, e.replaceRows(read, "own", nowNs))
			default:
				units := int64(c.Uint8()) % (rate.Limit + 2)
				requestFollowsGCRARule(t, e, rate, nowNs, units)
			}
			fresh, err := computeResetTimes(e.rows, e.resetTimes.current.startNs)
			require.NoError(t, err)
			require.Equal(t, fresh, e.resetTimes)
		}
	})
}

func storedValues(rows map[rowKey]StoredRow) []StoredRow {
	values := make([]StoredRow, 0, len(rows))
	for _, r := range rows {
		values = append(values, r)
	}
	slices.SortFunc(values, func(left, right StoredRow) int {
		return cmp.Or(cmp.Compare(left.BucketStartNs, right.BucketStartNs), cmp.Compare(left.PodID, right.PodID))
	})
	return values
}

// requestFollowsGCRARule checks one request against the GCRA rule and against
// the formula over all rows, then applies it.
func requestFollowsGCRARule(t *testing.T, e *entry, rate Rate, nowNs, units int64) {
	t.Helper()
	before := resetAtByFormula(e.rows)
	rowsBefore := len(e.rows)

	result, err := e.check(nowNs, rate, units)
	require.NoError(t, err)
	require.Equal(t, before, e.resetTimes.resetAtNs)

	consumedNs := ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)
	gcraAfter := max(before, nowNs) + consumedNs
	allowed := units <= rate.Limit && gcraAfter-nowNs <= rate.Duration.Nanoseconds()+roundingSlackNsOf(rate)
	require.Equal(t, allowed, result.allowed)
	if !allowed || units == 0 {
		require.Equal(t, before, result.resetAtNs)
		require.Zero(t, result.consumedNs)
		require.Equal(t, remainingBySearch(rate, nowNs, before), result.remaining)
		e.apply(result, "own")
		require.Equal(t, before, resetAtByFormula(e.rows))
		require.Len(t, e.rows, rowsBefore)
		return
	}

	require.Equal(t, consumedNs, result.consumedNs)
	e.apply(result, "own")
	after := resetAtByFormula(e.rows)
	require.Equal(t, after, result.resetAtNs)
	require.LessOrEqual(t, after, gcraAfter, "never charges more than GCRA")
	require.GreaterOrEqual(t, after, before)
	require.Equal(t, remainingBySearch(rate, nowNs, after), result.remaining)
	ownKey := rowKey{bucketStartNs: bucketStartNs(nowNs, rate.bucketWidthNs()), podID: "own"}
	own := e.rows[ownKey]
	require.True(t, own.own)
	// As if a flush landed, so `forgetLostUnits` keeps the units once the
	// bucket is past its last write.
	own.committedNs, own.committedResetAtNs = own.consumedNs, own.resetAtNs
	e.rows[ownKey] = own
}

// resetAtByFormula computes `reset_at` straight from its definition, in
// O(n²).
func resetAtByFormula(rows map[rowKey]row) int64 {
	var resetAtNs int64
	for key := range rows {
		bucketResetAtNs := key.bucketStartNs
		var newerConsumedNs int64
		for other, r := range rows {
			switch {
			case r.consumedNs == 0:
			case other.bucketStartNs == key.bucketStartNs:
				bucketResetAtNs += r.consumedNs
			case other.bucketStartNs > key.bucketStartNs:
				newerConsumedNs += r.consumedNs
			}
		}
		for other, r := range rows {
			if other.bucketStartNs == key.bucketStartNs && r.consumedNs > 0 {
				bucketResetAtNs = max(bucketResetAtNs, r.resetAtNs)
			}
		}
		if rows[key].consumedNs > 0 {
			resetAtNs = max(resetAtNs, bucketResetAtNs+newerConsumedNs)
		}
	}
	return resetAtNs
}

// remainingBySearch returns the largest number of units that fit at `nowNs`.
func remainingBySearch(rate Rate, nowNs, resetAtNs int64) int64 {
	for units := rate.Limit; units > 0; units-- {
		if max(resetAtNs, nowNs)+ceilDiv(units*rate.Duration.Nanoseconds(), rate.Limit)-nowNs <= rate.Duration.Nanoseconds()+roundingSlackNsOf(rate) {
			return units
		}
	}
	return 0
}

// roundingSlackNsOf is the rounding slack beyond `Duration`: below the cost of
// one unit, and at most `Limit - 1` ns, the most that rounding adds to
// `Limit` requests.
func roundingSlackNsOf(rate Rate) int64 {
	return min(ceilDiv(rate.Duration.Nanoseconds(), rate.Limit), rate.Limit) - 1
}

func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}

// timedRequest is one allowed request of plain GCRA: `consumedNs` at
// `atNs` on a pod.
type timedRequest struct {
	atNs       int64
	consumedNs int64
	pod        int
}

// plainResetAtNs is the `reset_at` of plain GCRA on one machine after all
// requests, in time order: `max over requests i of (time of i + consumed of
// requests at or after i)`.
func plainResetAtNs(requests []timedRequest) int64 {
	sorted := slices.SortedStableFunc(slices.Values(requests), func(left, right timedRequest) int {
		return cmp.Compare(left.atNs, right.atNs)
	})
	var resetAtNs, laterNs int64
	for i := len(sorted) - 1; i >= 0; i-- {
		laterNs += sorted[i].consumedNs
		resetAtNs = max(resetAtNs, sorted[i].atNs+laterNs)
	}
	return resetAtNs
}

// rowsOf returns the rows that pods write for the requests: per bucket and
// pod, the consumed time and the GCRA `reset_at` of that pod's requests.
func rowsOf(requests []timedRequest, widthNs int64) map[rowKey]row {
	rows := make(map[rowKey]row)
	for _, r := range slices.SortedStableFunc(slices.Values(requests), func(left, right timedRequest) int {
		return cmp.Compare(left.atNs, right.atNs)
	}) {
		key := rowKey{bucketStartNs: bucketStartNs(r.atNs, widthNs), podID: fmt.Sprintf("pod_%d", r.pod)}
		stored := rows[key]
		stored.consumedNs += r.consumedNs
		stored.resetAtNs = max(stored.resetAtNs, r.atNs) + r.consumedNs
		rows[key] = stored
	}
	return rows
}

// TestBucketsLoseTheOrderOfPeerRequests shows why no per-row summary of a
// bucket can match plain GCRA exactly. Pod B writes the same row in both
// histories: 4 ns consumed and a `reset_at` of 110. Pod A writes the same
// row too. Plain GCRA still differs, because it depends on how many of B's
// units come after A's request, which the rows do not record. The formula
// charges less than both, by less than the bucket width.
func TestBucketsLoseTheOrderOfPeerRequests(t *testing.T) {
	const widthNs = 10
	podA := timedRequest{atNs: 107, consumedNs: 20, pod: 0}
	first := []timedRequest{podA, {atNs: 100, consumedNs: 2, pod: 1}, {atNs: 108, consumedNs: 2, pod: 1}}
	second := []timedRequest{podA, {atNs: 100, consumedNs: 2, pod: 1}, {atNs: 106, consumedNs: 1, pod: 1}, {atNs: 109, consumedNs: 1, pod: 1}}

	require.Equal(t, rowsOf(first, widthNs), rowsOf(second, widthNs))
	require.Equal(t, int64(129), plainResetAtNs(first))
	require.Equal(t, int64(128), plainResetAtNs(second))
	require.Equal(t, int64(127), resetAtByFormula(rowsOf(first, widthNs)))
}

// FuzzBucketsStayWithinOneWidthOfPlainGCRA guarantees how the buckets compare
// with plain GCRA over the same requests from several pods: `reset_at` is
// never later, so the limiter never charges more, and earlier by less than
// one bucket width. It is exact when one pod wrote each bucket.
func FuzzBucketsStayWithinOneWidthOfPlainGCRA(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		widthNs := int64(c.Uint8()%50) + 1
		podCount := int(c.Uint8()%3) + 1
		var requests []timedRequest
		var atNs int64 = 1_000
		for range int(c.Uint8() % 40) {
			atNs += int64(c.Uint8() % 16)
			requests = append(requests, timedRequest{atNs: atNs, consumedNs: int64(c.Uint8()%30) + 1, pod: int(c.Uint8()) % podCount})
		}
		rows := rowsOf(requests, widthNs)
		got := resetAtByFormula(rows)
		times, err := computeResetTimes(rows, math.MaxInt64)
		require.NoError(t, err)
		require.Equal(t, got, times.resetAtNs)

		plain := plainResetAtNs(requests)
		require.LessOrEqual(t, got, plain, "never charges more than plain GCRA")
		require.Less(t, plain-got, max(widthNs, 1), "earlier by less than one bucket width")

		podsPerBucket := map[int64]map[string]bool{}
		for key := range rows {
			if podsPerBucket[key.bucketStartNs] == nil {
				podsPerBucket[key.bucketStartNs] = map[string]bool{}
			}
			podsPerBucket[key.bucketStartNs][key.podID] = true
		}
		onePodPerBucket := true
		for _, pods := range podsPerBucket {
			onePodPerBucket = onePodPerBucket && len(pods) == 1
		}
		if onePodPerBucket && len(requests) > 0 {
			require.Equal(t, plain, got, "exact when one pod wrote each bucket")
		}
	})
}
