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
	start := testClock.Now()
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 3}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	_, err = l.Allow(ctx, Request{Fingerprint: fingerprint("b"), Rate: request.Rate, Units: 0})
	require.NoError(t, err)

	require.NoError(t, l.flush(ctx))
	require.Equal(t, []StoredRow{{
		Fingerprint:    request.Fingerprint,
		BucketStartNs:  start.UnixNano(),
		PodID:          l.podID,
		ConsumedNs:     (3 * time.Second).Nanoseconds(),
		ResetAtNs:      start.Add(3 * time.Second).UnixNano(),
		CompactAfterNs: start.Add(time.Second + maxWriteDelay + maxClockSkew).UnixNano(),
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
	require.Equal(t, (5 * time.Second).Nanoseconds(), rows[0].ConsumedNs)
	require.Equal(t, start.Add(5*time.Second).UnixNano(), rows[0].ResetAtNs)
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
	require.Equal(t, time.Second.Nanoseconds(), store.storedRows(request.Fingerprint)[0].ConsumedNs)

	store.setFailWrites(true)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Error(t, l.flush(ctx))
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)

	store.setFailWrites(false)
	require.NoError(t, l.flush(ctx))
	require.Equal(t, (3 * time.Second).Nanoseconds(), store.storedRows(request.Fingerprint)[0].ConsumedNs)
}

// TestFlushStopsWritingABucketAfterTheLastWrite guarantees that a pod never
// writes a bucket after `maxWriteDelay` past its end. MySQL may refuse the
// write from then on, by a clock up to `maxClockSkew` ahead. Units that are
// already free by then are dropped.
func TestFlushStopsWritingABucketAfterTheLastWrite(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	start := testClock.Now()
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 10, Duration: 10 * time.Second}, Units: 1}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)

	store.setFailWrites(true)
	testClock.Set(start.Add(time.Second + maxWriteDelay))
	require.Error(t, l.flush(ctx))
	require.Equal(t, 1, store.writeCallCount())

	testClock.Set(start.Add(time.Second + maxWriteDelay + time.Nanosecond))
	require.NoError(t, l.flush(ctx))
	require.Equal(t, 1, store.writeCallCount())
	require.False(t, hasEntry(l, request.Fingerprint))
}

// TestLimiterKeepsLimitingWhileWritesFail guarantees that a pod whose writes
// all fail admits exactly what it admits when they succeed. Its units then
// exist only in its memory, so forgetting them would open the limit.
func TestLimiterKeepsLimitingWhileWritesFail(t *testing.T) {
	for _, rate := range []Rate{{Limit: 5, Duration: time.Hour}, {Limit: 10, Duration: time.Minute}} {
		t.Run(fmt.Sprintf("%d per %s", rate.Limit, rate.Duration), func(t *testing.T) {
			allowedCount := func(failWrites bool) int {
				ctx := context.Background()
				l, testClock := newTestLimiter(t)
				var background []func()
				l.runInBackground = func(f func()) { background = append(background, f) }
				memoryStoreOf(t, l).setFailWrites(failWrites)
				request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
				allowed := 0
				for range 2 * rate.Duration / time.Second {
					result, err := l.Allow(ctx, request)
					require.NoError(t, err)
					if result.Allowed {
						allowed++
					}
					for _, f := range background {
						f()
					}
					background = nil
					testClock.Tick(time.Second)
					err = l.flush(ctx)
					if !failWrites {
						require.NoError(t, err)
					}
				}
				memoryStoreOf(t, l).setFailWrites(false)
				return allowed
			}
			working := allowedCount(false)
			require.Greater(t, working, int(rate.Limit))
			require.Equal(t, working, allowedCount(true))
		})
	}
}

// TestFlushRemovesAnUnusedEntryBeforeItsResetAt guarantees that a limit with
// a long duration does not keep its entry in memory until `reset_at`, and
// that the next request reads the units back and decides the same.
func TestFlushRemovesAnUnusedEntryBeforeItsResetAt(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	request := Request{Fingerprint: fingerprint("a"), Rate: Rate{Limit: 1, Duration: 30 * 24 * time.Hour}, Units: 1}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	widthNs := request.Rate.bucketWidthNs()
	lastWriteAt := time.Unix(0, lastWriteAtNs(bucketStartNs(testClock.Now().UnixNano(), widthNs), widthNs))

	testClock.Set(lastWriteAt)
	require.NoError(t, l.flush(ctx))
	require.True(t, hasEntry(l, request.Fingerprint), "the pod still writes its row")
	testClock.Set(lastWriteAt.Add(time.Nanosecond))
	require.NoError(t, l.flush(ctx))
	require.False(t, hasEntry(l, request.Fingerprint))

	result, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.False(t, result.Allowed)
}

// TestFlushKeepsAnEntryWhileItsRowsAreWritable guarantees that an idle entry
// stays while the pod may still write its row. A new entry would read the row
// from a replica, which may lag, and lose units to GREATEST.
func TestFlushKeepsAnEntryWhileItsRowsAreWritable(t *testing.T) {
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
	require.True(t, hasEntry(l, idle.Fingerprint), "the row is idle and written, but still writable")

	testClock.Tick(time.Second + time.Nanosecond)
	require.NoError(t, l.flush(ctx))
	require.False(t, hasEntry(l, idle.Fingerprint))
	require.True(t, hasEntry(l, busy.Fingerprint))

	result, err := l.Allow(ctx, idle)
	require.NoError(t, err)
	require.Equal(t, Result{Allowed: true, Remaining: 9, ResetAt: testClock.Now().Add(time.Second)}, result)
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

// TestFlushWritesRowsByBucketNotByRequest is the regression test for the
// session design, which wrote a new row on almost every allowed request of a
// limit under its limit. Now a pod writes at most one row per bucket per
// flush: `⌈flushInterval / width⌉ + 1` rows per limit per second, the same for
// 10 and for 1,000 requests per second.
func TestFlushWritesRowsByBucketNotByRequest(t *testing.T) {
	ctx := context.Background()
	const seconds = 20
	for _, duration := range []time.Duration{time.Second, 10 * time.Second, time.Minute, time.Hour} {
		rate := Rate{Limit: 1_000_000_000, Duration: duration}
		widthNs := rate.bucketWidthNs()
		perFlushMax := int((flushInterval.Nanoseconds()+widthNs-1)/widthNs) + 1
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
					require.LessOrEqual(t, len(store.writeCalls[writes]), perFlushMax)
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

// TestManyPodsKeepRowsAndReadsBounded is the regression test for the row
// growth that a review found: with 100 ms buckets, every busy pod kept about
// 71 open rows per limit, and at 28 pods reads passed their bound and failed
// open. 50 pods now send a request every 100 ms to one 1 s limit for
// a minute, and flush, read and compact every second. A read must stay a
// few buckets long, and the table must hold at most the open and not yet
// compacted buckets of every pod.
func TestManyPodsKeepRowsAndReadsBounded(t *testing.T) {
	ctx := context.Background()
	const pods = 50
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	store := newMemoryStore(testClock)
	var background []func()
	limiters := make([]*limiter, pods)
	for i := range limiters {
		l, err := newLimiter(Config{Clock: testClock, Store: store, Random: rand.New(rand.NewPCG(uint64(i), 1))})
		require.NoError(t, err)
		l.runInBackground = func(f func()) { background = append(background, f) }
		limiters[i] = l
	}
	request := Request{Fingerprint: fingerprint("hot"), Rate: Rate{Limit: 1_000_000, Duration: time.Second}, Units: 1}
	// 1 s buckets stay open for 1 s + 2 s + 5 s, so at most 9 are open, and
	// at most 20 closed ones wait for a compaction.
	const bucketsMax = 29

	var rowsMax, bucketsReadMax int
	for second := range 60 {
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
		for len(background) > 0 {
			work := background[0]
			background = background[1:]
			work()
		}
		if second < 20 {
			continue
		}
		rowsMax = max(rowsMax, len(store.storedRows(request.Fingerprint)))
		rows, err := store.ReadRows(ctx, request.Fingerprint, limiters[0].podID, maxBucketsPerRead)
		require.NoError(t, err)
		buckets := map[int64]bool{}
		for _, r := range rows {
			buckets[r.BucketStartNs] = true
		}
		bucketsReadMax = max(bucketsReadMax, len(buckets))
	}
	t.Logf("at most %d stored rows and %d buckets per read", rowsMax, bucketsReadMax)
	require.LessOrEqual(t, bucketsReadMax, bucketsMax)
	require.LessOrEqual(t, rowsMax, pods*bucketsMax)
}
