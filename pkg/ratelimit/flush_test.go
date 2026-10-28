package ratelimit

import (
	"context"
	"encoding/binary"
	"fmt"
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

func TestFlushWritesOnlyRowsWithNewUnits(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 3}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("b"), Rate: rate, Units: 0})
	require.NoError(t, err)

	nowNs := testClock.Now().UnixNano()
	periodNs := periodLengthNs(rate.Duration)
	startNs := periodStartNs(nowNs, periodNs)
	sum := 3 * growthSince(nowNs, startNs, rate.Duration)
	require.NoError(t, l.flush(ctx))
	require.Equal(t, []StoredRow{{
		Fingerprint:   request.Fingerprint,
		PeriodStartNs: startNs,
		PodID:         l.podID,
		ScaledSum:     sum,
		DeleteAfterNs: deleteAfterNs(startNs, periodNs, rate.Duration, sum),
	}}, store.storedRows(request.Fingerprint))
	require.Empty(t, store.storedRows(fingerprint("b")))

	require.NoError(t, l.flush(ctx))
	require.Equal(t, 1, store.writeCallCount())

	request.Units = 2
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))
	rows := store.storedRows(request.Fingerprint)
	require.Len(t, rows, 1)
	require.InEpsilon(t, 5*growthSince(nowNs, startNs, rate.Duration), rows[0].ScaledSum, 1e-15)
}

// TestFlushRetriesAFailedWriteWithTheLatestSum guarantees that a lost write
// costs nothing: the next flush sends the latest sum, which holds its units.
func TestFlushRetriesAFailedWriteWithTheLatestSum(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	rate := Rate{Limit: 10, Duration: 10 * time.Second}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
	nowNs := testClock.Now().UnixNano()
	unitSum := growthSince(nowNs, periodStartNs(nowNs, periodLengthNs(rate.Duration)), rate.Duration)

	store.setFailWrites(true)
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.Error(t, l.flush(ctx))
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)

	store.setFailWrites(false)
	require.NoError(t, l.flush(ctx))
	require.InEpsilon(t, 2*unitSum, store.storedRows(request.Fingerprint)[0].ScaledSum, 1e-15)
}

// TestFlushKeepsAnEntryWhileItHasARowInTheCurrentPeriod guarantees that an
// idle entry stays until its period ends. A new entry would read its row
// from a replica that may lag and lose units to GREATEST. Afterwards the next
// request reads the units back and decides the same.
func TestFlushKeepsAnEntryWhileItHasARowInTheCurrentPeriod(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	rate := Rate{Limit: 10, Duration: time.Second}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 10}
	periodNs := periodLengthNs(rate.Duration)
	testClock.Set(time.Unix(0, periodStartNs(testClock.Now().UnixNano(), periodNs)+periodNs-2*time.Second.Nanoseconds()))
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)

	testClock.Tick(time.Second)
	require.NoError(t, l.flush(ctx))
	require.True(t, hasEntry(l, request.Fingerprint), "the pod has a row in the current period")

	testClock.Tick(time.Second)
	require.NoError(t, l.flush(ctx))
	require.True(t, hasEntry(l, request.Fingerprint), "the level of 1.35 units is not idle")

	testClock.Tick(time.Second)
	require.NoError(t, l.flush(ctx))
	require.False(t, hasEntry(l, request.Fingerprint))

	result, err := l.Allow(ctx, Request{Fingerprint: request.Fingerprint, Rate: rate, Units: 0})
	require.NoError(t, err)
	require.Equal(t, rate.Limit, result.Remaining)
}

// TestFlushRemovesAnUnusedEntry guarantees that a limit with a long duration
// leaves memory once its period ends and nobody asks for it, even though its
// level is far from zero, and that the next request reads the units back.
func TestFlushRemovesAnUnusedEntry(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	rate := Rate{Limit: 2, Duration: time.Hour}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 2}
	periodNs := periodLengthNs(rate.Duration)
	testClock.Set(time.Unix(0, periodStartNs(testClock.Now().UnixNano(), periodNs)+periodNs-time.Second.Nanoseconds()))
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))

	testClock.Tick(rowsStaleFor)
	require.NoError(t, l.flush(ctx))
	require.False(t, hasEntry(l, request.Fingerprint))

	result, err := l.Allow(ctx, Request{Fingerprint: request.Fingerprint, Rate: rate, Units: 1})
	require.NoError(t, err)
	require.False(t, result.Allowed)
}

func TestFlushSplitsWritesIntoCallsOfMaxRowsPerWrite(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	for i := range maxRowsPerWrite + 1 {
		var name [32]byte
		binary.BigEndian.PutUint64(name[:], uint64(i))
		_, err := l.Allow(ctx, Request{Fingerprint: name, Rate: Rate{Limit: 1, Duration: time.Hour}, Units: 1})
		require.NoError(t, err)
	}
	require.NoError(t, l.flush(ctx))
	require.Len(t, store.writeCalls, 2)
	require.Len(t, store.writeCalls[0], maxRowsPerWrite)
	require.Len(t, store.writeCalls[1], 1)
}

// TestFlushWritesOneRowPerLimitNotPerRequest guarantees that a pod writes one
// row per active limit per flush, two when a period ends between flushes, the
// same for 10 and for 1,000 requests per second.
func TestFlushWritesOneRowPerLimitNotPerRequest(t *testing.T) {
	ctx := context.Background()
	const seconds = 30
	for _, duration := range []time.Duration{time.Second, time.Minute} {
		rate := Rate{Limit: 1_000_000_000, Duration: duration}
		rowsWritten := map[int]int{}
		for _, requestsPerSecond := range []int{10, 1_000} {
			t.Run(fmt.Sprintf("%s at %d per second", duration, requestsPerSecond), func(t *testing.T) {
				l, testClock := newTestLimiter(t)
				store := memoryStoreOf(t, l)
				request := Request{Fingerprint: fingerprint("under the limit"), Rate: rate, Units: 1}
				for range seconds {
					for range requestsPerSecond {
						result, err := l.Allow(ctx, request)
						require.NoError(t, err)
						require.True(t, result.Allowed)
						testClock.Tick(time.Second / time.Duration(requestsPerSecond))
					}
					writes := store.writeCallCount()
					require.NoError(t, l.flush(ctx))
					require.Equal(t, writes+1, store.writeCallCount())
					require.LessOrEqual(t, len(store.writeCalls[writes]), 2)
					rowsWritten[requestsPerSecond] += len(store.writeCalls[writes])
				}
			})
		}
		require.Equal(t, rowsWritten[10], rowsWritten[1_000], "%s: rows written must not depend on the request rate", duration)
	}
}

func TestLimiterFlushesEverySecondAndOnClose(t *testing.T) {
	ctx := context.Background()
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	store := newMemoryStore(testClock)
	l, err := New(Config{Clock: testClock, Store: store, Random: rand.New(rand.NewPCG(1, 2))})
	require.NoError(t, err)
	rate := Rate{Limit: 10, Duration: time.Hour}

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("ticked"), Rate: rate, Units: 1})
	require.NoError(t, err)
	testClock.Tick(time.Second)
	require.Eventually(t, func() bool { return len(store.storedRows(fingerprint("ticked"))) == 1 }, 5*time.Second, time.Millisecond)

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("closed"), Rate: rate, Units: 1})
	require.NoError(t, err)
	require.NoError(t, l.Close())
	require.Len(t, store.storedRows(fingerprint("closed")), 1)
	require.NoError(t, l.Close())

	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("closed"), Rate: rate, Units: 1})
	require.Error(t, err)
}

// TestManyPodsKeepRowsAndReadsBounded runs 50 pods that send a request every
// 100 ms to one 1 s limit for five minutes, and flush, read and clean up
// every second. A read must stay a few periods long, and the table must hold
// at most a few periods of every pod.
func TestManyPodsKeepRowsAndReadsBounded(t *testing.T) {
	ctx := context.Background()
	const pods = 50
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	store := newMemoryStore(testClock)
	limiters := make([]*limiter, pods)
	for i := range limiters {
		limiters[i] = newTestLimiterOn(t, testClock, store, uint64(i))
	}
	request := Request{Fingerprint: fingerprint("hot"), Rate: Rate{Limit: 1_000_000, Duration: time.Second}, Units: 1}
	// A 10 s period with a sum of about 10 × e^10 fades about 21 s after it
	// starts, plus `maxClockSkew`, so at most 4 periods have rows.
	const periodsMax = 4

	var rowsMax, periodsReadMax int
	for second := range 300 {
		for range 10 {
			for _, l := range limiters {
				result, err := l.Allow(ctx, request)
				require.NoError(t, err)
				require.True(t, result.Allowed)
			}
			testClock.Tick(100 * time.Millisecond)
		}
		for _, l := range limiters {
			require.NoError(t, l.flush(ctx))
		}
		require.NoError(t, limiters[second%pods].cleanUp(ctx))
		rowsMax = max(rowsMax, len(store.storedRows(request.Fingerprint)))
		rows, err := store.ReadRows(ctx, request.Fingerprint, limiters[0].podID, maxPeriodsPerRead)
		require.NoError(t, err)
		periods := map[int64]bool{}
		for _, r := range rows {
			periods[r.PeriodStartNs] = true
		}
		periodsReadMax = max(periodsReadMax, len(periods))
	}
	t.Logf("at most %d stored rows and %d periods per read", rowsMax, periodsReadMax)
	require.LessOrEqual(t, periodsReadMax, periodsMax)
	require.LessOrEqual(t, rowsMax, pods*periodsMax)
}
