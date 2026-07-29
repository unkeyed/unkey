package cache_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestSWR_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockClock := clock.NewTestClock()

	c, err := cache.New(cache.Config[string, string]{
		Fresh:    1 * time.Minute,
		Stale:    5 * time.Minute,
		MaxSize:  100,
		Resource: "test",
		Clock:    mockClock,
	})
	require.NoError(t, err)

	t.Run("miss on first call", func(t *testing.T) {
		value, hit, err := c.SWR(ctx, "key1", func(ctx context.Context) (string, error) {
			return "value1", nil
		}, func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "value1", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("hit on subsequent call within fresh time", func(t *testing.T) {
		// First call to populate cache
		_, _, err := c.SWR(ctx, "key2", func(ctx context.Context) (string, error) {
			return "value2", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})
		require.NoError(t, err)

		// Second call should hit cache
		value, hit, err := c.SWR(ctx, "key2", func(ctx context.Context) (string, error) {
			t.Fatal("should not call refresh function")
			return "", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "value2", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("null cache hit", func(t *testing.T) {
		// First call returns not found error
		_, _, err := c.SWR(ctx, "key3", func(ctx context.Context) (string, error) {
			return "", sql.ErrNoRows
		}, func(err error) cache.Op {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}
			return cache.Noop
		})
		require.Error(t, err)
		require.True(t, db.IsNotFound(err))

		// Second call should return null hit
		value, hit, err := c.SWR(ctx, "key3", func(ctx context.Context) (string, error) {
			t.Fatal("should not call refresh function")
			return "", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "", value)
		require.Equal(t, cache.Null, hit)
	})

	t.Run("stale hit returns cached value", func(t *testing.T) {
		// First call to populate cache
		_, _, err := c.SWR(ctx, "key4", func(ctx context.Context) (string, error) {
			return "value4", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})
		require.NoError(t, err)

		// Move time forward past fresh but within stale
		mockClock.Tick(2 * time.Minute)

		// Should return cached value with hit status
		value, hit, err := c.SWR(ctx, "key4", func(ctx context.Context) (string, error) {
			// This will be called in background
			return "updated_value4", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "value4", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("miss after stale time", func(t *testing.T) {
		// First call to populate cache
		_, _, err := c.SWR(ctx, "key5", func(ctx context.Context) (string, error) {
			return "value5", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})
		require.NoError(t, err)

		// Move time forward past stale
		mockClock.Tick(6 * time.Minute)

		// Should call refresh and return new value
		value, hit, err := c.SWR(ctx, "key5", func(ctx context.Context) (string, error) {
			return "new_value5", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "new_value5", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("error returns miss", func(t *testing.T) {
		expectedErr := errors.New("refresh error")
		value, hit, err := c.SWR(ctx, "key6", func(ctx context.Context) (string, error) {
			return "", expectedErr
		}, func(err error) cache.Op {
			return cache.Noop
		})

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
		require.Equal(t, "", value)
		require.Equal(t, cache.Miss, hit)
	})
}

func TestSWR_CoalescesConcurrentMisses(t *testing.T) {
	const callers = 100

	ctx := context.Background()
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "miss_singleflight_test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})

	originStarted := make(chan struct{})
	var originCalls atomic.Int32
	load := func(context.Context) (string, error) {
		if originCalls.Add(1) == 1 {
			close(originStarted)
		}
		<-release
		return "value", nil
	}

	type result struct {
		value string
		hit   cache.CacheHit
		err   error
	}
	results := make(chan result, callers)
	start := make(chan struct{})
	for range callers {
		go func() {
			<-start
			value, hit, loadErr := c.SWR(ctx, "key", load, func(error) cache.Op { return cache.WriteValue })
			results <- result{value: value, hit: hit, err: loadErr}
		}()
	}
	close(start)

	select {
	case <-originStarted:
	case <-time.After(time.Second):
		t.Fatal("origin load did not start")
	}

	require.Never(t, func() bool {
		return originCalls.Load() > 1
	}, 100*time.Millisecond, time.Millisecond, "concurrent misses must share one origin load")
	releaseOnce.Do(func() { close(release) })

	for range callers {
		res := <-results
		require.NoError(t, res.err)
		require.Equal(t, cache.Hit, res.hit)
		require.Equal(t, "value", res.value)
	}
	require.Equal(t, int32(1), originCalls.Load())
}

func TestSWR_ServesLastKnownGoodBeyondTenMinutes(t *testing.T) {
	ctx := context.Background()
	mockClock := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    10 * time.Second,
		Stale:    24 * time.Hour,
		MaxSize:  10,
		Resource: "last_known_good_test",
		Clock:    mockClock,
	})
	require.NoError(t, err)

	c.Set(ctx, "key", "last-known-good")
	mockClock.Tick(11 * time.Minute)

	refreshAttempted := make(chan struct{})
	originErr := errors.New("database unavailable")
	value, hit, err := c.SWR(ctx, "key", func(context.Context) (string, error) {
		close(refreshAttempted)
		return "", originErr
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	require.NoError(t, err)
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "last-known-good", value)

	select {
	case <-refreshAttempted:
	case <-time.After(time.Second):
		t.Fatal("background refresh was not attempted")
	}

	value, hit = c.Get(ctx, "key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "last-known-good", value, "a failed refresh must not discard last-known-good data")
}

func TestSWR_RevalidationQueueSaturationDoesNotBlock(t *testing.T) {
	const (
		revalidationWorkers = 10
		revalidationQueue   = 1000
	)

	ctx := context.Background()
	mockClock := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Second,
		Stale:    time.Hour,
		MaxSize:  revalidationWorkers + revalidationQueue + 1,
		Resource: "saturation_test",
		Clock:    mockClock,
	})
	require.NoError(t, err)

	releaseWorkers := make(chan struct{})
	workersStarted := make(chan struct{})
	var started atomic.Int32
	for i := range revalidationWorkers {
		key := fmt.Sprintf("worker-%d", i)
		c.Set(ctx, key, "stale")
	}
	for i := range revalidationQueue {
		c.Set(ctx, fmt.Sprintf("queued-%d", i), "stale")
	}
	droppedKey := "dropped"
	c.Set(ctx, droppedKey, "stale")
	mockClock.Tick(2 * time.Second)

	for i := range revalidationWorkers {
		key := fmt.Sprintf("worker-%d", i)
		value, hit, swrErr := c.SWR(ctx, key, func(context.Context) (string, error) {
			if started.Add(1) == revalidationWorkers {
				close(workersStarted)
			}
			<-releaseWorkers
			return "fresh", nil
		}, func(error) cache.Op { return cache.WriteValue })
		require.NoError(t, swrErr)
		require.Equal(t, cache.Hit, hit)
		require.Equal(t, "stale", value)
	}

	select {
	case <-workersStarted:
	case <-time.After(time.Second):
		close(releaseWorkers)
		t.Fatal("revalidation workers did not start")
	}

	var queuedRefreshes atomic.Int32
	for i := range revalidationQueue {
		key := fmt.Sprintf("queued-%d", i)
		value, hit, swrErr := c.SWR(ctx, key, func(context.Context) (string, error) {
			queuedRefreshes.Add(1)
			return "fresh", nil
		}, func(error) cache.Op { return cache.WriteValue })
		require.NoError(t, swrErr)
		require.Equal(t, cache.Hit, hit)
		require.Equal(t, "stale", value)
	}

	type swrResult struct {
		value string
		hit   cache.CacheHit
		err   error
	}
	firstCallDone := make(chan swrResult, 1)
	var droppedRefreshes atomic.Int32
	go func() {
		value, hit, swrErr := c.SWR(ctx, droppedKey, func(context.Context) (string, error) {
			droppedRefreshes.Add(1)
			return "fresh", nil
		}, func(error) cache.Op { return cache.WriteValue })
		firstCallDone <- swrResult{value: value, hit: hit, err: swrErr}
	}()

	select {
	case result := <-firstCallDone:
		require.NoError(t, result.err)
		require.Equal(t, cache.Hit, result.hit)
		require.Equal(t, "stale", result.value)
	case <-time.After(100 * time.Millisecond):
		close(releaseWorkers)
		t.Fatal("stale cache hit blocked on a saturated revalidation queue")
	}
	require.Zero(t, droppedRefreshes.Load())

	close(releaseWorkers)
	require.Eventually(t, func() bool {
		return queuedRefreshes.Load() == revalidationQueue
	}, time.Second, time.Millisecond)

	_, _, err = c.SWR(ctx, droppedKey, func(context.Context) (string, error) {
		droppedRefreshes.Add(1)
		return "fresh", nil
	}, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return droppedRefreshes.Load() == 1
	}, time.Second, time.Millisecond, "a dropped revalidation must be eligible for a later retry")
}

func TestSWR_RevalidationIsDeduplicatedBeforeEnqueue(t *testing.T) {
	const blockedWorkers = 9

	ctx := context.Background()
	mockClock := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Second,
		Stale:    time.Hour,
		MaxSize:  100,
		Resource: "deduplication_test",
		Clock:    mockClock,
	})
	require.NoError(t, err)

	for i := range blockedWorkers {
		c.Set(ctx, fmt.Sprintf("blocker-%d", i), "stale")
	}
	c.Set(ctx, "target", "stale")
	c.Set(ctx, "sentinel", "stale")
	mockClock.Tick(2 * time.Second)

	releaseBlockers := make(chan struct{})
	blockersStarted := make(chan struct{})
	var blockerCount atomic.Int32
	for i := range blockedWorkers {
		key := fmt.Sprintf("blocker-%d", i)
		_, _, err = c.SWR(ctx, key, func(context.Context) (string, error) {
			if blockerCount.Add(1) == blockedWorkers {
				close(blockersStarted)
			}
			<-releaseBlockers
			return "fresh", nil
		}, func(error) cache.Op { return cache.WriteValue })
		require.NoError(t, err)
	}

	select {
	case <-blockersStarted:
	case <-time.After(time.Second):
		close(releaseBlockers)
		t.Fatal("revalidation workers did not start")
	}

	releaseTarget := make(chan struct{})
	targetStarted := make(chan struct{})
	var targetRefreshes atomic.Int32
	targetRefresh := func(context.Context) (string, error) {
		if targetRefreshes.Add(1) == 1 {
			close(targetStarted)
			<-releaseTarget
		}
		return "fresh", nil
	}

	_, _, err = c.SWR(ctx, "target", targetRefresh, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)
	select {
	case <-targetStarted:
	case <-time.After(time.Second):
		close(releaseTarget)
		close(releaseBlockers)
		t.Fatal("target revalidation did not start")
	}

	for range 100 {
		value, hit, swrErr := c.SWR(ctx, "target", targetRefresh, func(error) cache.Op { return cache.WriteValue })
		require.NoError(t, swrErr)
		require.Equal(t, cache.Hit, hit)
		require.Equal(t, "stale", value)
	}

	sentinelRefreshed := make(chan struct{})
	_, _, err = c.SWR(ctx, "sentinel", func(context.Context) (string, error) {
		close(sentinelRefreshed)
		return "fresh", nil
	}, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)

	close(releaseTarget)
	select {
	case <-sentinelRefreshed:
	case <-time.After(time.Second):
		close(releaseBlockers)
		t.Fatal("revalidation queue did not drain")
	}

	require.Equal(t, int32(1), targetRefreshes.Load())
	close(releaseBlockers)
}
