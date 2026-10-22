package ratelimit

import (
	"context"
	"encoding/binary"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

func memoryStoreOf(t *testing.T, l *limiter) *memoryStore {
	t.Helper()
	store, ok := l.store.(*memoryStore)
	require.True(t, ok)
	return store
}

func hasEntry(l *limiter, fingerprint [32]byte) bool {
	s := &l.shards[shardIndex(fingerprint)]
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.entries[fingerprint]
	return ok
}

func TestFlushWritesOnlySessionsWithNewUnits(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	start := testClock.Now()
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 3}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("b"), Rate: request.Rate, Units: 0})
	require.NoError(t, err)

	require.NoError(t, l.flush(ctx))
	sessions := store.sessions(request.Fingerprint)
	require.Len(t, sessions, 1)
	require.Equal(t, StoredSession{
		Fingerprint:    request.Fingerprint,
		SessionID:      sessions[0].SessionID,
		AnchorAtNs:     start.UnixNano(),
		ConsumedNs:     (3 * time.Second).Nanoseconds(),
		CompactAfterMs: start.Add(30 * time.Minute).UnixMilli(),
	}, sessions[0])
	require.Empty(t, store.sessions(fingerprint("b")))

	require.NoError(t, l.flush(ctx))
	require.Equal(t, 1, store.writeCallCount())

	request.Units = 2
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))
	require.Equal(t, (5 * time.Second).Nanoseconds(), store.sessions(request.Fingerprint)[0].ConsumedNs)
	require.Len(t, store.writeCalls[1], 1)
}

func TestFlushRetriesAFailedWriteWithTheLatestValue(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}

	store.setFailWrites(true)
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Error(t, l.flush(ctx))

	store.setFailWrites(false)
	require.NoError(t, l.flush(ctx))
	require.Equal(t, time.Second.Nanoseconds(), store.sessions(request.Fingerprint)[0].ConsumedNs)

	store.setFailWrites(true)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Error(t, l.flush(ctx))
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)

	store.setFailWrites(false)
	require.NoError(t, l.flush(ctx))
	require.Equal(t, (3 * time.Second).Nanoseconds(), store.sessions(request.Fingerprint)[0].ConsumedNs)
}

// TestFlushWritesAFrozenSessionThenForgetsIt uses 1 unit per second, so the
// lifetime ends the first session while `reset_at` is still ahead.
func TestFlushWritesAFrozenSessionThenForgetsIt(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 3600, Duration: time.Hour}, Units: 1800}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	e := l.shards[shardIndex(request.Fingerprint)].entries[request.Fingerprint]
	frozenID := e.liveID

	testClock.Set(time.Unix(0, e.liveSessionEndsAtNs))
	request.Units = 1
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))

	sessions := store.sessions(request.Fingerprint)
	require.Len(t, sessions, 2)
	require.Equal(t, frozenID, sessions[1].SessionID)
	require.Equal(t, (1800 * time.Second).Nanoseconds(), sessions[1].ConsumedNs)

	require.NoError(t, l.flush(ctx))
	require.Equal(t, map[string]int64{e.liveID: time.Second.Nanoseconds()}, e.writtenNs)
}

// TestFlushDropsUnitsAtCompactAfter guarantees that a pod never writes a
// session at or after its `compact_after`, when cleanup may already have
// merged the row.
func TestFlushDropsUnitsAtCompactAfter(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	start := testClock.Now()
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: time.Hour}, Units: 1}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)

	store.setFailWrites(true)
	testClock.Set(start.Add(30*time.Minute - time.Nanosecond))
	require.Error(t, l.flush(ctx))
	require.Equal(t, 1, store.writeCallCount())

	testClock.Set(start.Add(30 * time.Minute))
	require.NoError(t, l.flush(ctx))
	require.Equal(t, 1, store.writeCallCount())
	require.False(t, hasEntry(l, request.Fingerprint))
}

func TestFlushRemovesAnEntryOnlyWhenItIsIdleAndWritten(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	idle := Request{Fingerprint: fingerprint("idle"), Rate: rate, Units: 1}
	busy := Request{Fingerprint: fingerprint("busy"), Rate: rate, Units: 10}
	_, err := l.AllowMany(ctx, []Request{idle, busy})
	require.NoError(t, err)

	store.setFailWrites(true)
	testClock.Tick(2 * time.Second)
	require.Error(t, l.flush(ctx))
	require.True(t, hasEntry(l, idle.Fingerprint))

	store.setFailWrites(false)
	require.NoError(t, l.flush(ctx))
	require.NoError(t, l.flush(ctx))
	require.False(t, hasEntry(l, idle.Fingerprint))
	require.True(t, hasEntry(l, busy.Fingerprint))

	result, err := l.Allow(ctx, idle)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 9, ResetAt: testClock.Now().Add(time.Second)}, result)
}

func TestFlushWritesAtMost10000SessionsPerCall(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	for i := range 10_001 {
		var name [32]byte
		binary.BigEndian.PutUint64(name[:], uint64(i))
		_, err := l.Allow(ctx, Request{Fingerprint: name, Rate: Rate{Limit: 1, Duration: time.Hour}, Units: 1})
		require.NoError(t, err)
	}
	require.NoError(t, l.flush(ctx))
	require.Len(t, store.writeCalls, 2)
	require.Len(t, store.writeCalls[0], 10_000)
	require.Len(t, store.writeCalls[1], 1)
}

func TestLimiterFlushesEverySecondAndOnClose(t *testing.T) {
	ctx := context.Background()
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	store := newMemoryStore()
	l, err := New(Config{Clock: testClock, Store: store, Random: rand.New(rand.NewPCG(1, 2))})
	require.NoError(t, err)
	rate := Rate{Limit: 10, Duration: time.Hour}

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("ticked"), Rate: rate, Units: 1})
	require.NoError(t, err)
	testClock.Tick(time.Second)
	require.Eventually(t, func() bool { return len(store.sessions(fingerprint("ticked"))) == 1 }, 5*time.Second, time.Millisecond)

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("closed"), Rate: rate, Units: 1})
	require.NoError(t, err)
	require.NoError(t, l.Close())
	require.Len(t, store.sessions(fingerprint("closed")), 1)
	require.NoError(t, l.Close())

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("closed"), Rate: rate, Units: 1})
	require.Error(t, err)
}
