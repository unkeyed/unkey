package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestRatelimit_FailsOverToLocalWhenOriginErrors asserts that when the origin
// counter returns errors, Ratelimit still returns a decision based on local
// state rather than failing the request.
func TestRatelimit_FailsOverToLocalWhenOriginErrors(t *testing.T) {
	t.Parallel()

	origin := newFailingCounter(errors.New("origin is down"))
	svc, err := New(Config{
		Clock: clock.NewTestClock(), Counter: origin, DB: newTestDB(t), Region: "test-region"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	resp, err := svc.Ratelimit(context.Background(), RatelimitRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		Namespace:   "ns",
		Identifier:  "id",
		Limit:       10,
		Duration:    time.Minute,
		Cost:        1,
	})
	require.NoError(t, err, "Ratelimit must not propagate origin errors to callers")
	require.True(t, resp.Success, "local state was empty, so the request should pass on local-only info")
}

// TestFetchFromOrigin_CircuitBreakerShortCircuitsAfterTrip asserts that a
// sustained stream of origin failures trips the circuit breaker and
// subsequent requests bypass the origin entirely, rather than each paying
// the full Redis timeout. We verify this by measuring that far fewer origin
// calls are made after the trip than would be expected without it.
func TestFetchFromOrigin_CircuitBreakerShortCircuitsAfterTrip(t *testing.T) {
	t.Parallel()

	origin := newFailingCounter(errors.New("origin is down"))
	svc, err := New(Config{
		Clock: clock.NewTestClock(), Counter: origin, DB: newTestDB(t), Region: "test-region"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	ws := uid.New(uid.WorkspacePrefix)

	// Drive a burst of cold-window requests. Each unique identifier triggers
	// origin GET for two windows (current + previous).
	const warmupRequests = 100
	for i := range warmupRequests {
		_, _ = svc.Ratelimit(context.Background(), RatelimitRequest{
			WorkspaceID: ws,
			Namespace:   "ns",
			Identifier:  fmt.Sprintf("warmup-%d", i),
			Limit:       10,
			Duration:    time.Minute,
			Cost:        1,
		})
	}
	callsAfterWarmup := origin.getCalls.Load()
	require.Positive(t, callsAfterWarmup, "cold windows should attempt origin fetches")

	// Now send another burst with fresh identifiers. If the breaker has
	// tripped, most of these calls should be rejected by the breaker without
	// reaching the counter, so origin call count grows far less than 2×burst.
	const measuredRequests = 100
	for i := range measuredRequests {
		_, _ = svc.Ratelimit(context.Background(), RatelimitRequest{
			WorkspaceID: ws,
			Namespace:   "ns",
			Identifier:  fmt.Sprintf("measured-%d", i),
			Limit:       10,
			Duration:    time.Minute,
			Cost:        1,
		})
	}
	additionalCalls := origin.getCalls.Load() - callsAfterWarmup

	// Every request has two cold windows, so an open breaker would let through
	// 2 * measuredRequests calls. A tripped breaker should be far less.
	maxTolerated := int64(measuredRequests)
	require.Less(t, additionalCalls, maxTolerated,
		"circuit breaker should short-circuit most calls after repeated failures (got %d, tolerated <%d)",
		additionalCalls, maxTolerated)
}

func TestCounterEntryEnsureFreshFromOrigin_RefreshesWarmEntryAfterFreshUntil(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)
	fetchOps := []string{}
	var fetchCalls atomic.Int64
	entry := counterEntry{
		fetch: func(_ context.Context, op string) int64 {
			fetchOps = append(fetchOps, op)
			return fetchCalls.Add(1) * 10
		},
	}

	entry.EnsureFreshFromOrigin(ctx, start)
	require.Equal(t, int64(1), fetchCalls.Load(), "cold entry should fetch once")
	require.Equal(t, int64(10), entry.val.Load())

	freshUntil := start.Add(originFreshDuration)
	entry.EnsureFreshFromOrigin(ctx, freshUntil.Add(-time.Millisecond))
	require.Equal(t, int64(1), fetchCalls.Load(), "warm entry should not refresh before freshUntil")
	require.Equal(t, int64(10), entry.val.Load())

	entry.EnsureFreshFromOrigin(ctx, freshUntil)
	require.Equal(t, int64(2), fetchCalls.Load(), "warm entry should refresh at freshUntil")
	require.Equal(t, int64(20), entry.val.Load())
	require.Equal(t, []string{"fetch_cold", "fetch_stale"}, fetchOps)
}

func TestCounterEntryEnsureFreshFromOrigin_GatesConcurrentWarmRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)
	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})
	var closeFetchStarted sync.Once
	var fetchCalls atomic.Int64
	entry := counterEntry{
		fetch: func(context.Context, string) int64 {
			fetchCalls.Add(1)
			closeFetchStarted.Do(func() { close(fetchStarted) })
			<-releaseFetch
			return 10
		},
	}
	entry.hydrated.Store(true)
	entry.originFreshUntilMs.Store(start.Add(originFreshDuration).UnixMilli())

	const goroutines = 32
	ready := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			<-ready
			entry.EnsureFreshFromOrigin(ctx, start.Add(originFreshDuration))
		}()
	}

	close(ready)
	<-fetchStarted
	close(releaseFetch)
	wg.Wait()

	require.Equal(t, int64(1), fetchCalls.Load(), "only one stale warm caller should refresh origin")
	require.Equal(t, int64(10), entry.val.Load())
}

func TestCounterEntryEnsureFreshFromOrigin_FailedWarmRefreshSuppressesRetryUntilFreshUntil(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)
	var fetchCalls atomic.Int64
	entry := counterEntry{
		fetch: func(context.Context, string) int64 {
			if fetchCalls.Add(1) == 1 {
				return 0
			}
			return 99
		},
	}
	entry.val.Store(7)
	entry.hydrated.Store(true)
	entry.originFreshUntilMs.Store(start.Add(originFreshDuration).UnixMilli())

	failedRefresh := start.Add(originFreshDuration)
	entry.EnsureFreshFromOrigin(ctx, failedRefresh)
	require.Equal(t, int64(1), fetchCalls.Load())
	require.Equal(t, int64(7), entry.val.Load(), "failed fetch returns 0 and should preserve local state")
	require.Equal(t, failedRefresh.Add(originFreshDuration).UnixMilli(), entry.originFreshUntilMs.Load(), "failed refresh still advances freshUntil")

	entry.EnsureFreshFromOrigin(ctx, failedRefresh.Add(time.Second))
	require.Equal(t, int64(1), fetchCalls.Load(), "failed refresh should not retry before freshUntil")
	require.Equal(t, int64(7), entry.val.Load())

	entry.EnsureFreshFromOrigin(ctx, failedRefresh.Add(originFreshDuration))
	require.Equal(t, int64(2), fetchCalls.Load(), "failed refresh should retry at freshUntil")
	require.Equal(t, int64(99), entry.val.Load())
}
