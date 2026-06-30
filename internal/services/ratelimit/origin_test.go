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
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
)

// timeoutError is a net.Error whose Timeout() reports true, mimicking the
// socket-level i/o timeout the redis client surfaces when a deadline blows but
// the error does not unwrap to context.DeadlineExceeded.
type timeoutError struct{}

func (timeoutError) Error() string   { return "read tcp: i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// TestErrorReason asserts that errorReason keeps breaker short-circuits,
// real timeouts, and other errors in separate reason buckets — the count of
// circuit_open dwarfs the rest during a burst, so they must not collapse
// together.
func TestErrorReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"breaker tripped", circuitbreaker.ErrTripped, "circuit_open"},
		{"breaker half-open", circuitbreaker.ErrTooManyRequests, "circuit_open"},
		{"wrapped breaker trip", fmt.Errorf("fetch: %w", circuitbreaker.ErrTripped), "circuit_open"},
		{"context deadline", context.DeadlineExceeded, "timeout"},
		{"wrapped deadline", fmt.Errorf("get: %w", context.DeadlineExceeded), "timeout"},
		{"socket i/o timeout", timeoutError{}, "timeout"},
		{"redis error", errors.New("READONLY You can't write against a read only replica"), "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, errorReason(tt.err))
		})
	}
}

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

func TestFetchFromOrigin_ContextCanceledDoesNotTripCircuitBreaker(t *testing.T) {
	t.Parallel()

	origin := newFailingCounter(context.Canceled)
	svc := &service{
		origin:               origin,
		originCircuitBreaker: circuitbreaker.New[int64]("ratelimitOrigin"),
	}
	key := counterKey{
		workspaceID: "ws",
		namespace:   "ns",
		identifier:  "id",
		durationMs:  time.Minute.Milliseconds(),
		sequence:    1,
	}

	const attempts = 20
	for range attempts {
		count, ok := svc.fetchFromOrigin(context.Background(), key, "fetch_cold")
		require.False(t, ok)
		require.Zero(t, count)
	}

	require.Equal(t, int64(attempts), origin.getCalls.Load(),
		"context cancellations should not open the origin circuit breaker")
}

func TestCounterEntryEnsureFreshFromOrigin_RefreshesWarmEntryAfterFreshUntil(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)
	fetchOps := []string{}
	var fetchCalls atomic.Int64
	entry := counterEntry{
		fetch: func(_ context.Context, op string) (int64, bool) {
			fetchOps = append(fetchOps, op)
			return fetchCalls.Add(1) * 10, true
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
		fetch: func(context.Context, string) (int64, bool) {
			fetchCalls.Add(1)
			closeFetchStarted.Do(func() { close(fetchStarted) })
			<-releaseFetch
			return 10, true
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

func TestCounterEntryEnsureFreshFromOrigin_FailedColdRefreshRetriesBeforeFreshDuration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)
	var fetchCalls atomic.Int64
	entry := counterEntry{
		fetch: func(context.Context, string) (int64, bool) {
			if fetchCalls.Add(1) == 1 {
				return 0, false
			}
			return 99, true
		},
	}

	entry.EnsureFreshFromOrigin(ctx, start)
	require.Equal(t, int64(1), fetchCalls.Load())
	require.Equal(t, int64(0), entry.val.Load(), "failed cold fetch should preserve zero local state")
	require.Equal(t, start.Add(originFetchRetryDuration).UnixMilli(), entry.originFreshUntilMs.Load())

	entry.EnsureFreshFromOrigin(ctx, start.Add(originFetchRetryDuration-time.Millisecond))
	require.Equal(t, int64(1), fetchCalls.Load(), "failed cold fetch should suppress immediate retry")

	entry.EnsureFreshFromOrigin(ctx, start.Add(originFetchRetryDuration))
	require.Equal(t, int64(2), fetchCalls.Load(), "failed cold fetch should retry before full fresh duration")
	require.Equal(t, int64(99), entry.val.Load())
	require.Equal(t,
		start.Add(originFetchRetryDuration).Add(originFreshDuration).UnixMilli(),
		entry.originFreshUntilMs.Load(),
	)
}

func TestCounterEntryEnsureFreshFromOrigin_FailedWarmRefreshRetriesBeforeFreshDuration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)
	var fetchCalls atomic.Int64
	entry := counterEntry{
		fetch: func(context.Context, string) (int64, bool) {
			if fetchCalls.Add(1) == 1 {
				return 0, false
			}
			return 99, true
		},
	}
	entry.val.Store(7)
	entry.hydrated.Store(true)
	entry.originFreshUntilMs.Store(start.Add(originFreshDuration).UnixMilli())

	failedRefresh := start.Add(originFreshDuration)
	entry.EnsureFreshFromOrigin(ctx, failedRefresh)
	require.Equal(t, int64(1), fetchCalls.Load())
	require.Equal(t, int64(7), entry.val.Load(), "failed fetch should preserve local state")
	require.Equal(t,
		failedRefresh.Add(originFetchRetryDuration).UnixMilli(),
		entry.originFreshUntilMs.Load(),
		"failed refresh should only advance retry deadline",
	)

	entry.EnsureFreshFromOrigin(ctx, failedRefresh.Add(originFetchRetryDuration-time.Millisecond))
	require.Equal(t, int64(1), fetchCalls.Load(), "failed refresh should not retry before retry deadline")
	require.Equal(t, int64(7), entry.val.Load())

	entry.EnsureFreshFromOrigin(ctx, failedRefresh.Add(originFetchRetryDuration))
	require.Equal(t, int64(2), fetchCalls.Load(), "failed refresh should retry before full fresh duration")
	require.Equal(t, int64(99), entry.val.Load())
}
