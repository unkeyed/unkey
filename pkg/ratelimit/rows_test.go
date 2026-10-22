package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// writePeer writes a session of another pod that used `units` of `rate`.
func writePeer(t *testing.T, store *memoryStore, request Request, id string, anchor time.Time, units int64) {
	t.Helper()
	require.NoError(t, store.WriteSessions(context.Background(), []StoredSession{{
		Fingerprint:    request.Fingerprint,
		SessionID:      id,
		AnchorAtNs:     anchor.UnixNano(),
		ConsumedNs:     request.Rate.unitsToNs(units),
		CompactAfterMs: anchor.Add(compactAfter).UnixMilli(),
	}}))
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
// on the sessions this pod already has, and that a down database gets at
// most one waiting read per limit every `rowsStaleFor`.
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

	testClock.Tick(rowsStaleFor - time.Nanosecond)
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

func TestLimiterIgnoresAReadThatReachesTheRowLimit(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 100_000, Duration: time.Hour}, Units: 1}
	for i := range maxRowsPerRead {
		writePeer(t, store, request, fmt.Sprintf("peer%04d", i), testClock.Now().Add(-time.Duration(i)), 100)
	}

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 99_999, ResetAt: testClock.Now().Add(36 * time.Millisecond)}, result)
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
