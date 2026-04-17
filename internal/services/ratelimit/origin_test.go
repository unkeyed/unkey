package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

// TestRatelimit_FailsOverToLocalWhenOriginErrors asserts that when the origin
// counter returns errors, Ratelimit still returns a decision based on local
// state rather than failing the request.
func TestRatelimit_FailsOverToLocalWhenOriginErrors(t *testing.T) {
	t.Parallel()

	origin := newFailingCounter(errors.New("origin is down"))
	svc, err := New(Config{Clock: clock.NewTestClock(), Counter: origin})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	resp, err := svc.Ratelimit(context.Background(), RatelimitRequest{
		Name:       "ns",
		Identifier: "id",
		Limit:      10,
		Duration:   time.Minute,
		Cost:       1,
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
	svc, err := New(Config{Clock: clock.NewTestClock(), Counter: origin})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	// Drive a burst of cold-window requests. Each unique identifier triggers
	// origin GET for two windows (current + previous).
	const warmupRequests = 100
	for i := range warmupRequests {
		_, _ = svc.Ratelimit(context.Background(), RatelimitRequest{
			Name:       "ns",
			Identifier: fmt.Sprintf("warmup-%d", i),
			Limit:      10,
			Duration:   time.Minute,
			Cost:       1,
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
			Name:       "ns",
			Identifier: fmt.Sprintf("measured-%d", i),
			Limit:      10,
			Duration:   time.Minute,
			Cost:       1,
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
