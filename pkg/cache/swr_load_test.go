package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type swrCallResult struct {
	value string
	hit   CacheHit
	err   error
}

// TestSWRConcurrentMissesShareOneOriginRead guarantees that callers missing the
// same key at the same time cause one origin read, so a cold or expired hot key
// cannot stampede the origin, while a miss on another key is not held up.
func TestSWRConcurrentMissesShareOneOriginRead(t *testing.T) {
	for _, expired := range []bool{false, true} {
		name := "cold"
		if expired {
			name = "expired"
		}
		t.Run(name, func(t *testing.T) {
			c, clk := newPausedRefreshCache(t)
			if expired {
				c.Set(t.Context(), "key", "old")
				clk.Tick(2 * time.Hour)
			}

			var calls atomic.Int32
			release := make(chan struct{})
			origin := func(context.Context) (string, error) {
				calls.Add(1)
				<-release
				return "new", nil
			}
			op := func(error) Op { return WriteValue }

			const callers = 20
			results := make(chan swrCallResult, callers)
			for range callers {
				go func() {
					value, hit, err := c.SWR(context.Background(), "key", origin, op)
					results <- swrCallResult{value, hit, err}
				}()
			}
			waitForWaiters(t, c, "key", callers)

			value, hit, err := c.SWR(t.Context(), "other", func(context.Context) (string, error) {
				return "other-value", nil
			}, op)
			require.NoError(t, err)
			require.Equal(t, Hit, hit)
			require.Equal(t, "other-value", value)

			close(release)
			for range callers {
				require.Equal(t, swrCallResult{"new", Hit, nil}, <-results)
			}
			require.Equal(t, int32(1), calls.Load())

			value, hit, err = c.SWR(t.Context(), "key", origin, op)
			require.NoError(t, err)
			require.Equal(t, Hit, hit)
			require.Equal(t, "new", value)
			require.Equal(t, int32(1), calls.Load(), "the shared read must be published to the cache")
		})
	}
}

// TestSWRSharedReadFollowsCallerCancellation guarantees that a first caller
// that goes away, such as a disconnected client, does not fail the callers
// waiting for its origin read, while its deadline still bounds the read. A
// waiting caller that goes away stops waiting at once, and a caller that is
// already gone does not start a read at all.
func TestSWRSharedReadFollowsCallerCancellation(t *testing.T) {
	c, _ := newPausedRefreshCache(t)
	op := func(err error) Op {
		if err != nil {
			return Noop
		}
		return WriteValue
	}
	originCtx := make(chan context.Context, 1)
	release := make(chan struct{})
	origin := func(ctx context.Context) (string, error) {
		originCtx <- ctx
		select {
		case <-release:
			return "value", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	deadline := time.Now().Add(time.Hour)
	firstCtx, cancelFirst := context.WithDeadline(context.Background(), deadline)
	defer cancelFirst()
	results := make(chan swrCallResult, 2)
	go func() {
		value, hit, err := c.SWR(firstCtx, "key", origin, op)
		results <- swrCallResult{value, hit, err}
	}()
	readCtx := <-originCtx
	go func() {
		value, hit, err := c.SWR(context.Background(), "key", origin, op)
		results <- swrCallResult{value, hit, err}
	}()
	leavingCtx, cancelLeaving := context.WithCancel(context.Background())
	defer cancelLeaving()
	leaving := make(chan swrCallResult, 1)
	go func() {
		value, hit, err := c.SWR(leavingCtx, "key", origin, op)
		leaving <- swrCallResult{value, hit, err}
	}()
	waitForWaiters(t, c, "key", 3)

	cancelLeaving()
	select {
	case got := <-leaving:
		require.Equal(t, swrCallResult{"", Miss, context.Canceled}, got)
	case <-time.After(time.Second):
		t.Fatal("canceled waiting caller kept waiting for the shared read")
	}
	waitForWaiters(t, c, "key", 2)

	cancelFirst()
	waitForWaiters(t, c, "key", 1)
	require.NoError(t, readCtx.Err(), "the read must continue for the waiting caller")
	readDeadline, ok := readCtx.Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, readDeadline)

	close(release)
	for range 2 {
		require.Equal(t, swrCallResult{"value", Hit, nil}, <-results)
	}

	_, hit, err := c.SWR(firstCtx, "gone", func(context.Context) (string, error) {
		t.Error("an already canceled caller must not read the origin")
		return "", nil
	}, op)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, Miss, hit)

}

// TestSWRAbandonedReadIsReplaced guarantees that a read every caller has left
// is canceled and stops taking new callers, so a later caller starts a fresh
// read, and that the abandoned read finishing late does not detach callers
// from the fresh one.
func TestSWRAbandonedReadIsReplaced(t *testing.T) {
	c, _ := newPausedRefreshCache(t)
	op := func(err error) Op {
		if err != nil {
			return Noop
		}
		return WriteValue
	}
	var calls atomic.Int32
	abandonedCtx := make(chan context.Context, 1)
	releaseAbandoned := make(chan struct{})
	releaseFresh := make(chan struct{})
	origin := func(ctx context.Context) (string, error) {
		if calls.Add(1) == 1 {
			abandonedCtx <- ctx
			<-releaseAbandoned
			return "", ctx.Err()
		}
		<-releaseFresh
		return "fresh", nil
	}

	ctx, abandon := context.WithCancel(context.Background())
	first := make(chan swrCallResult, 1)
	go func() {
		value, hit, err := c.SWR(ctx, "key", origin, op)
		first <- swrCallResult{value, hit, err}
	}()
	readCtx := <-abandonedCtx
	abandon()
	select {
	case <-readCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("a read that every caller left was not canceled")
	}

	fresh := make(chan swrCallResult, 2)
	go func() {
		value, hit, err := c.SWR(context.Background(), "key", origin, op)
		fresh <- swrCallResult{value, hit, err}
	}()
	require.Eventually(t, func() bool { return calls.Load() == 2 }, 5*time.Second, time.Millisecond,
		"a caller after an abandoned read must start a fresh one")

	close(releaseAbandoned)
	got := <-first
	require.ErrorIs(t, got.err, context.Canceled)
	require.Equal(t, Miss, got.hit)

	go func() {
		value, hit, err := c.SWR(context.Background(), "key", origin, op)
		fresh <- swrCallResult{value, hit, err}
	}()
	waitForWaiters(t, c, "key", 2)
	close(releaseFresh)
	for range 2 {
		require.Equal(t, swrCallResult{"fresh", Hit, nil}, <-fresh)
	}
	require.Equal(t, int32(2), calls.Load())
}

// TestSWRExpiredReadJoinsRunningRefresh guarantees that a read that finds its
// entry expired while a background refresh for the key is running waits for
// that refresh instead of reading the origin a second time.
func TestSWRExpiredReadJoinsRunningRefresh(t *testing.T) {
	c, clk := newPausedRefreshCache(t)
	c.Set(t.Context(), "key", "stale")
	clk.Tick(2 * time.Minute)
	op := func(error) Op { return WriteValue }

	started := make(chan struct{})
	release := make(chan struct{})
	releaseRefresh := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseRefresh)
	value, _, err := c.SWR(t.Context(), "key", func(context.Context) (string, error) {
		close(started)
		<-release
		return "refreshed", nil
	}, op)
	require.NoError(t, err)
	require.Equal(t, "stale", value)
	refresh := <-c.revalidateC
	refreshDone := make(chan struct{})
	go func() {
		defer close(refreshDone)
		refresh()
	}()
	<-started

	clk.Tick(time.Hour)
	var expiredReadCalls atomic.Int32
	expired := make(chan swrCallResult, 1)
	go func() {
		value, hit, err := c.SWR(context.Background(), "key", func(context.Context) (string, error) {
			expiredReadCalls.Add(1)
			return "duplicate", nil
		}, op)
		expired <- swrCallResult{value, hit, err}
	}()
	waitForWaiters(t, c, "key", 2)

	releaseRefresh()
	require.Equal(t, swrCallResult{"refreshed", Hit, nil}, <-expired)
	require.Zero(t, expiredReadCalls.Load())
	<-refreshDone
}

// TestSWRSharedReadPanicDoesNotStrandWaiters guarantees that a panic in a
// shared origin read is raised in the caller that ran it, where request panic
// recovery can handle it, while the waiting callers get an error instead of
// blocking forever, and later callers can read the key again.
func TestSWRSharedReadPanicDoesNotStrandWaiters(t *testing.T) {
	c, _ := newPausedRefreshCache(t)
	op := func(error) Op { return WriteValue }
	release := make(chan struct{})
	origin := func(context.Context) (string, error) {
		<-release
		panic("origin exploded")
	}

	const callers = 3
	type outcome struct {
		recovered any
		hit       CacheHit
		err       error
	}
	outcomes := make(chan outcome, callers)
	for range callers {
		go func() {
			var hit CacheHit
			var err error
			defer func() { outcomes <- outcome{recover(), hit, err} }()
			_, hit, err = c.SWR(context.Background(), "key", origin, op)
		}()
	}
	waitForWaiters(t, c, "key", callers)
	close(release)

	panics := 0
	for range callers {
		got := <-outcomes
		if got.recovered != nil {
			panics++
			require.Equal(t, "origin exploded", got.recovered)
			continue
		}
		require.ErrorIs(t, got.err, errLoadPanicked)
		require.Equal(t, Miss, got.hit)
	}
	require.Equal(t, 1, panics)

	value, _, err := c.SWR(t.Context(), "key", func(context.Context) (string, error) {
		return "recovered", nil
	}, op)
	require.NoError(t, err)
	require.Equal(t, "recovered", value)
}

// waitForWaiters waits until n callers, the first one included, wait for the
// read of key.
func waitForWaiters(t *testing.T, c *cache[string, string], key string, n int) {
	t.Helper()
	require.Eventually(t, func() bool {
		c.loadsMu.Lock()
		defer c.loadsMu.Unlock()
		f, ok := c.loads[key]
		return ok && f.waiters == n
	}, 5*time.Second, time.Millisecond)
}
