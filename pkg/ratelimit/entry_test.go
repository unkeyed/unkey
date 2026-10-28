package ratelimit

import (
	"maps"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestEntryReplaceRowsKeepsOwnUnitsUntilTheLastWrite guarantees that a stale
// replica never moves this pod's own row back while the pod still writes it,
// that peer rows are taken as read, and that the read decides once the pod
// no longer writes the row. The units that the store never acknowledged then
// move to the current row as far as they still reach, so they are not lost.
func TestEntryReplaceRowsKeepsOwnUnitsUntilTheLastWrite(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	widthNs := rate.bucketWidthNs()
	startNs := int64(time.Hour)
	e := newEntry(rate.Duration)
	result, err := e.check(startNs, rate, 5)
	require.NoError(t, err)
	e.apply(result, "own")
	ownKey := rowKey{bucketStartNs: startNs, podID: "own"}
	peerKey := rowKey{bucketStartNs: startNs, podID: "peer"}
	stale := []StoredRow{
		{BucketStartNs: startNs, PodID: "own", ConsumedNs: time.Second.Nanoseconds(), ResetAtNs: startNs + time.Second.Nanoseconds()},
		{BucketStartNs: startNs, PodID: "peer", ConsumedNs: 2 * time.Second.Nanoseconds(), ResetAtNs: startNs + 2*time.Second.Nanoseconds()},
	}

	require.NoError(t, e.replaceRows(stale, "own", startNs))
	require.Equal(t, row{consumedNs: 5 * time.Second.Nanoseconds(), resetAtNs: startNs + 5*time.Second.Nanoseconds(), own: true, committedNs: time.Second.Nanoseconds(), committedResetAtNs: startNs + time.Second.Nanoseconds()}, e.rows[ownKey])
	require.Equal(t, row{consumedNs: 2 * time.Second.Nanoseconds(), resetAtNs: startNs + 2*time.Second.Nanoseconds(), own: false, committedNs: 0, committedResetAtNs: 0}, e.rows[peerKey])
	require.Equal(t, startNs+7*time.Second.Nanoseconds(), e.resetTimes.resetAtNs)

	require.NoError(t, e.replaceRows(nil, "own", lastWriteAtNs(startNs, widthNs)))
	require.Contains(t, e.rows, ownKey, "a write may still be on its way")
	require.NotContains(t, e.rows, peerKey)

	afterLastWriteNs := lastWriteAtNs(startNs, widthNs) + 1
	currentStartNs := bucketStartNs(afterLastWriteNs, widthNs)
	carriedKey := rowKey{bucketStartNs: currentStartNs, podID: "own"}
	carried := row{consumedNs: startNs + 5*time.Second.Nanoseconds() - currentStartNs, resetAtNs: startNs + 5*time.Second.Nanoseconds(), own: true, committedNs: 0, committedResetAtNs: 0}
	require.NoError(t, e.replaceRows(stale[:1], "own", afterLastWriteNs))
	require.Equal(t, map[rowKey]row{
		ownKey:     {consumedNs: time.Second.Nanoseconds(), resetAtNs: startNs + time.Second.Nanoseconds(), own: false, committedNs: 0, committedResetAtNs: 0},
		carriedKey: carried,
	}, e.rows)
	require.Equal(t, startNs+5*time.Second.Nanoseconds(), e.resetTimes.resetAtNs, "the carried units end when the lost ones did")

	require.NoError(t, e.replaceRows(nil, "own", afterLastWriteNs))
	require.Equal(t, map[rowKey]row{carriedKey: carried}, e.rows, "cleanup may have merged the row into the base row")
}

// TestEntryReplaceRowsIsAllOrNothing guarantees that invalid rows change
// nothing, so a corrupt read cannot open the limit.
func TestEntryReplaceRowsIsAllOrNothing(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	nowNs := int64(time.Hour)
	e := newEntry(rate.Duration)
	result, err := e.check(nowNs, rate, 5)
	require.NoError(t, err)
	e.apply(result, "own")
	before := *e
	before.rows = map[rowKey]row{}
	for key, r := range e.rows {
		before.rows[key] = r
	}

	for name, rows := range map[string][]StoredRow{
		"repeated row": {
			{BucketStartNs: nowNs, PodID: "peer", ConsumedNs: 1, ResetAtNs: nowNs + 1},
			{BucketStartNs: nowNs, PodID: "peer", ConsumedNs: 2, ResetAtNs: nowNs + 2},
		},
		"negative consumed time": {{BucketStartNs: nowNs, PodID: "peer", ConsumedNs: -1, ResetAtNs: nowNs}},
		"sum beyond int64":       {{BucketStartNs: nowNs, PodID: "peer", ConsumedNs: math.MaxInt64, ResetAtNs: nowNs}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, e.replaceRows(rows, "own", nowNs))
			require.Equal(t, before, *e)
		})
	}
}

// TestEntryCheckMovesToTheBucketOfNow guarantees that a request after a
// bucket edge goes into the new bucket and counts every older one.
func TestEntryCheckMovesToTheBucketOfNow(t *testing.T) {
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	secondNs := time.Second.Nanoseconds()
	startNs := int64(time.Hour)
	e := newEntry(rate.Duration)
	for i, at := range []int64{0, secondNs / 2, secondNs, 3 * secondNs} {
		result, err := e.check(startNs+at, rate, 2)
		require.NoError(t, err)
		require.True(t, result.allowed, "request %d", i)
		e.apply(result, "own")
	}
	require.Len(t, e.rows, 3)
	require.Equal(t, startNs+8*secondNs, e.resetTimes.resetAtNs)
	require.Equal(t, row{consumedNs: 2 * secondNs, resetAtNs: startNs + 5*secondNs, own: true, committedNs: 0, committedResetAtNs: 0}, e.rows[rowKey{bucketStartNs: startNs + 3*secondNs, podID: "own"}])
}

// FuzzCarryLostUnitsNeverChargesMore guarantees that moving lost units to
// the current row never raises `reset_at` above what the rows give with the
// lost units in place, as if their writes had landed. This holds over many
// moves and after a later request. With peer rows that the pod has not seen,
// each move may add 1 ns.
func FuzzCarryLostUnitsNeverChargesMore(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		rate := Rate{Limit: 10, Duration: time.Duration(1+c.Uint8()%60) * time.Second}
		widthNs := rate.bucketWidthNs()
		stepNs := widthNs / 8
		e := newEntry(rate.Duration)
		landed := map[rowKey]row{}
		unseen := map[rowKey]row{}
		resetAtWith := func(rows ...map[rowKey]row) int64 {
			all := map[rowKey]row{}
			for _, more := range rows {
				maps.Copy(all, more)
			}
			times, err := computeResetTimes(all, math.MaxInt64)
			require.NoError(t, err)
			return times.resetAtNs
		}
		nowNs := time.Hour.Nanoseconds()
		// Each move may charge 1 ns more than the lost units still reach,
		// like the rounding of `unitsToNs`, but only to rows it has not seen.
		var carriesNs int64
		for range c.Uint8() % 100 {
			nowNs += int64(c.Uint8()%16) * stepNs
			startNs := bucketStartNs(nowNs, widthNs)
			consumedNs := int64(1+c.Uint8()%32) * stepNs
			switch c.Uint8() % 5 {
			case 0, 1:
				key := rowKey{bucketStartNs: startNs, podID: "own"}
				for _, rows := range []map[rowKey]row{e.rows, landed} {
					r := rows[key]
					r.own = true
					r.resetAtNs = max(r.resetAtNs, nowNs) + consumedNs
					r.consumedNs += consumedNs
					rows[key] = r
				}
				e.watchLostUnits(key, e.rows[key])
			case 2:
				key := rowKey{bucketStartNs: startNs, podID: "own"}
				if r, ok := e.rows[key]; ok {
					r.committedNs, r.committedResetAtNs = r.consumedNs, r.resetAtNs
					e.rows[key] = r
				}
			case 3:
				// A peer whose clock is ahead writes buckets after now.
				key := rowKey{bucketStartNs: startNs + int64(c.Uint8()%3)*widthNs, podID: "seen"}
				r := e.rows[key]
				r.resetAtNs = max(r.resetAtNs, key.bucketStartNs) + consumedNs
				r.consumedNs += consumedNs
				e.rows[key], landed[key] = r, r
			default:
				key := rowKey{bucketStartNs: startNs + int64(c.Uint8()%3)*widthNs, podID: "unseen"}
				r := unseen[key]
				r.resetAtNs = max(r.resetAtNs, key.bucketStartNs) + consumedNs
				r.consumedNs += consumedNs
				unseen[key] = r
			}
			require.NoError(t, e.carryLostUnits(nowNs))
			carriesNs++
			require.LessOrEqual(t, resetAtWith(e.rows), resetAtWith(landed))
			require.LessOrEqual(t, resetAtWith(e.rows, unseen), resetAtWith(landed, unseen)+carriesNs)
		}
		laterNs := nowNs + int64(c.Uint8()%64)*stepNs
		later := map[rowKey]row{{bucketStartNs: bucketStartNs(laterNs, widthNs), podID: "later"}: {consumedNs: stepNs, resetAtNs: laterNs + stepNs, own: false, committedNs: 0, committedResetAtNs: 0}}
		require.LessOrEqual(t, resetAtWith(e.rows, unseen, later), resetAtWith(landed, unseen, later)+carriesNs)
	})
}
