package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// writePeer stores the row of another pod that used `units` of `rate` at
// `at`.
func writePeer(t *testing.T, store *memoryStore, request Request, podID string, at time.Time, units int64) {
	t.Helper()
	widthNs := request.Rate.bucketWidthNs()
	startNs := bucketStartNs(at.UnixNano(), widthNs)
	consumedNs := request.Rate.unitsToNs(units)
	store.seed([]StoredRow{{
		Fingerprint:    request.Fingerprint,
		BucketStartNs:  startNs,
		PodID:          podID,
		ConsumedNs:     consumedNs,
		ResetAtNs:      at.UnixNano() + consumedNs,
		CompactAfterNs: compactAfterNs(startNs, widthNs),
	}})
}

func TestLimiterCountsPeerRows(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}
	writePeer(t, store, request, "peer", testClock.Now(), 9)

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 0, ResetAt: testClock.Now().Add(10 * time.Second)}, result)
	result, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.False(t, result.Allowed)
}

// TestLimiterFailsOpenWhenTheReadFails guarantees that a failed read decides
// on the rows this pod already has, and that a down database gets at most
// one waiting read per limit every `rowsStaleFor`.
func TestLimiterFailsOpenWhenTheReadFails(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	var background []func()
	l.runInBackground = func(f func()) { background = append(background, f) }
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 4}
	writePeer(t, store, request, "peer", testClock.Now(), 10)
	store.setFailReads(true)

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 6, ResetAt: testClock.Now().Add(4 * time.Second)}, result)
	require.Equal(t, 1, store.readCallCount())
	require.NoError(t, l.flush(ctx))

	testClock.Tick(rowsStaleFor - time.Nanosecond)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))
	require.Len(t, background, 1)
	background[0]()
	require.Equal(t, 2, store.readCallCount())

	store.setFailReads(false)
	testClock.Tick(rowsStaleFor)
	result, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, 3, store.readCallCount())
}

// TestLimiterIgnoresAReadThatReachesTheBucketLimit guarantees that a read
// whose oldest buckets may be missing decides on what the pod already has.
// A bucket that holds only a refused write does not count towards the bound.
func TestLimiterIgnoresAReadThatReachesTheBucketLimit(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	l.runInBackground = func(func()) {}
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 100_000, Duration: time.Hour}, Units: 1}
	widthNs := request.Rate.bucketWidthNs()
	for i := range maxBucketsPerRead {
		writePeer(t, store, request, "peer", testClock.Now().Add(-time.Duration(int64(i)*widthNs)), 100)
	}
	refusedStartNs := bucketStartNs(testClock.Now().UnixNano(), widthNs) + widthNs
	store.seed([]StoredRow{{
		Fingerprint:    request.Fingerprint,
		BucketStartNs:  refusedStartNs,
		PodID:          "refused",
		ResetAtNs:      refusedStartNs,
		CompactAfterNs: compactAfterNs(refusedStartNs, widthNs),
	}})

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 99_999, ResetAt: testClock.Now().Add(36 * time.Millisecond)}, result)
}

// TestLimiterSkipsReadsWhileTooManyRun guarantees that a pod starts no read
// while `readsInFlightMax` reads run, so new identifiers cannot take the
// whole replica pool, and that it reads again once a slot is free.
func TestLimiterSkipsReadsWhileTooManyRun(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}
	writePeer(t, store, request, "peer", testClock.Now(), 10)
	for range readsInFlightMax {
		l.readSlots <- struct{}{}
	}

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.True(t, result.Allowed, "the pod decides on what it knows")
	require.Zero(t, store.readCallCount())

	for range readsInFlightMax {
		<-l.readSlots
	}
	testClock.Tick(rowsStaleFor)
	writePeer(t, store, request, "peer", testClock.Now(), 10)
	result, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, 1, store.readCallCount())
}

// TestLimiterServesStaleRowsWhileItRefreshes walks a limit through fresh,
// stale, and expired peer rows.
func TestLimiterServesStaleRowsWhileItRefreshes(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	var background []func()
	l.runInBackground = func(f func()) { background = append(background, f) }
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}

	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 1, store.readCallCount())
	writePeer(t, store, request, "peer", testClock.Now(), 9)

	testClock.Tick(rowsFreshFor - time.Nanosecond)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Empty(t, background)

	testClock.Tick(time.Nanosecond)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Len(t, background, 1, "one background read per limit")
	require.Equal(t, 1, store.readCallCount())

	background[0]()
	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.False(t, result.Allowed)
}

// TestLimiterWaitsForARunningReadWhenRowsExpire starts a background read and
// lets the rows expire before it finishes. The next request must wait for
// that read, not decide without the peer rows.
func TestLimiterWaitsForARunningReadWhenRowsExpire(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	var background []func()
	l.runInBackground = func(f func()) { background = append(background, f) }
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Hour}, Units: 1}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	writePeer(t, store, request, "peer", testClock.Now(), 9)

	testClock.Tick(rowsFreshFor)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Len(t, background, 1)
	require.NoError(t, l.flush(ctx))

	testClock.Tick(rowsStaleFor)
	done := make(chan Result)
	go func() {
		result, allowErr := l.Allow(ctx, request)
		require.NoError(t, allowErr)
		done <- result
	}()
	time.Sleep(50 * time.Millisecond)
	background[0]()
	result := <-done
	require.False(t, result.Allowed)
	require.Equal(t, 2, store.readCallCount())
}

func TestLimiterAllowManyReadsPeerRowsOfEveryLimit(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	first := Request{Fingerprint: fingerprint("first"), Rate: rate, Units: 1}
	second := Request{Fingerprint: fingerprint("second"), Rate: rate, Units: 1}
	writePeer(t, store, first, "peer-first", testClock.Now(), 10)
	writePeer(t, store, second, "peer-second", testClock.Now(), 10)

	results, err := l.AllowMany(ctx, []Request{first, second})
	require.NoError(t, err)
	require.False(t, results[0].Allowed)
	require.False(t, results[1].Allowed)
	require.Equal(t, 2, store.readCallCount())
}

// TestLimiterNeverWritesPeerRows guarantees that a pod writes only its own
// rows. Writing a peer's row back could bring it back after cleanup merged
// it, and count its units twice.
func TestLimiterNeverWritesPeerRows(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}
	writePeer(t, store, request, "peer", testClock.Now(), 3)
	writes := store.writeCallCount()

	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))

	require.Equal(t, writes+1, store.writeCallCount())
	require.Len(t, store.writeCalls[writes], 1)
	require.Equal(t, l.podID, store.writeCalls[writes][0].PodID)
}

// TestLimiterCompactsAFingerprintWithManyClosedBuckets guarantees that a busy
// limit does not wait for a cleanup pass to reach it: a read that finds
// enough closed buckets merges them, and `reset_at` does not change.
func TestLimiterCompactsAFingerprintWithManyClosedBuckets(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	var background []func()
	l.runInBackground = func(f func()) { background = append(background, f) }
	request := Request{Fingerprint: fingerprint("busy"), Rate: Rate{Limit: 100, Duration: time.Hour}, Units: 0}
	const closedBuckets = minClosedBucketsPerCompaction + maxCompactionJitter
	widthNs := request.Rate.bucketWidthNs()
	newestAt := testClock.Now().Add(-10 * time.Minute)
	for i := range closedBuckets {
		writePeer(t, store, request, fmt.Sprintf("peer%d", i%3), newestAt.Add(-time.Duration(int64(i)*widthNs)), 1)
	}
	// Each unit is 36 s. The oldest request's candidate counts all of them.
	oldestAt := newestAt.Add(-time.Duration(int64(closedBuckets-1) * widthNs))
	wantResetAtNs := oldestAt.Add(closedBuckets * 36 * time.Second).UnixNano()

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, time.Unix(0, wantResetAtNs), result.ResetAt)
	require.Len(t, background, 1)
	background[0]()

	require.Equal(t, []StoredRow{{
		Fingerprint:    request.Fingerprint,
		BucketStartNs:  0,
		PodID:          "",
		ConsumedNs:     (closedBuckets * 36 * time.Second).Nanoseconds(),
		ResetAtNs:      wantResetAtNs,
		CompactAfterNs: wantResetAtNs + compactAfterNs(0, request.Rate.bucketWidthNs()),
	}}, store.storedRows(request.Fingerprint))

	testClock.Tick(rowsStaleFor)
	result, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, time.Unix(0, wantResetAtNs), result.ResetAt)
	require.Len(t, background, 1, "one base row starts no compaction")
}
