package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// writePeer stores the row of another pod that used `units` of `rate` at
// `at`.
func writePeer(t *testing.T, store *memoryStore, request Request, podID string, at time.Time, units int64) {
	t.Helper()
	periodMs := periodLengthMs(request.Rate.Duration)
	startMs := periodStartMs(at.UnixMilli(), periodMs)
	sum := float64(units) * growthSince(at.UnixMilli(), startMs, request.Rate.Duration)
	for _, stored := range store.storedRows(request.Fingerprint) {
		if stored.PeriodStartMs == startMs && stored.PodID == podID {
			sum += stored.ScaledSum
		}
	}
	store.seed([]StoredRow{{
		Fingerprint:   request.Fingerprint,
		PeriodStartMs: startMs,
		PodID:         podID,
		ScaledSum:     sum,
		DeleteAfterMs: deleteAfterMs(startMs, periodMs, request.Rate.Duration, sum),
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
	require.Equal(t, Result{Allowed: true, Remaining: 0, ResetAt: testClock.Now().Add(afterMs(request.Rate.Duration, 10, 9))}, result)
	result, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.False(t, result.Allowed)
}

// TestLimiterReadsPeersOfEveryPeriod guarantees that a read sums the rows of
// all periods, each decayed from its own start.
func TestLimiterReadsPeersOfEveryPeriod(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	rate := Rate{Limit: 100, Duration: time.Second}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 0}
	periodMs := periodLengthMs(rate.Duration)
	now := time.UnixMilli(periodStartMs(testClock.Now().UnixMilli(), periodMs) + time.Second.Milliseconds())
	testClock.Set(now)
	writePeer(t, store, request, "a", now.Add(-2*time.Second), 50)
	writePeer(t, store, request, "b", now.Add(-time.Second), 50)
	writePeer(t, store, request, "b", now, 30)

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	level := 50*0.1353352832366127 + 50*0.36787944117144233 + 30
	require.Equal(t, int64(100-level), result.Remaining)
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
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Hour}, Units: 4}
	writePeer(t, store, request, "peer", testClock.Now(), 10)
	store.setFailReads(true)

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 6, ResetAt: testClock.Now()}, result)
	require.Equal(t, 1, store.readCallCount())

	testClock.Tick(rowsStaleFor - time.Millisecond)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
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

// TestLimiterSkipsReadsWhileTooManyRun guarantees that a pod starts no read
// while `readsInFlightMax` reads run, so new identifiers cannot take the
// whole replica pool, and that it reads again once a slot is free.
func TestLimiterSkipsReadsWhileTooManyRun(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Hour}, Units: 1}
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
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Hour}, Units: 1}

	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 1, store.readCallCount())
	writePeer(t, store, request, "peer", testClock.Now(), 9)

	testClock.Tick(rowsFreshFor - time.Millisecond)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Empty(t, background)

	testClock.Tick(time.Millisecond)
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

// TestLimiterKeepsItsOwnUnitsThatTheReplicaLacks guarantees that a read
// from a lagging replica never makes a pod forget units it admitted.
func TestLimiterKeepsItsOwnUnitsThatTheReplicaLacks(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Hour}, Units: 10}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))
	store.mu.Lock()
	clear(store.rows)
	store.mu.Unlock()

	testClock.Tick(rowsStaleFor)
	result, err := l.Allow(ctx, Request{Fingerprint: request.Fingerprint, Rate: request.Rate, Units: 1})
	require.NoError(t, err)
	require.False(t, result.Allowed)
}

// TestLimiterNeverWritesPeerRows guarantees that a pod writes only its own
// rows. Writing a peer's summed row back would count its units twice.
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

func TestEntryReplaceRowsRejectsRowsOfOtherPods(t *testing.T) {
	e := newEntry(time.Second)
	err := e.replaceRows([]StoredRow{{PeriodStartMs: 0, PodID: "other", ScaledSum: 1}}, "own", time.Second.Milliseconds())
	require.Error(t, err)
	err = e.replaceRows([]StoredRow{{PeriodStartMs: 0, PodID: "", ScaledSum: 1}, {PeriodStartMs: 0, PodID: "", ScaledSum: 2}}, "own", time.Second.Milliseconds())
	require.Error(t, err)
	require.Empty(t, e.rows, "a rejected read changes nothing")
}
