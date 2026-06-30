package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

var errTestDownstream = errors.New("downstream test error")

func TestCircuitBreakerStates(t *testing.T) {

	c := clock.NewTestClock()
	cb := New[int]("test", WithCyclicPeriod(5*time.Second), WithClock(c), WithTripThreshold(3))

	// Test Closed State
	for i := 0; i < 3; i++ {
		_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
			return 0, errTestDownstream
		})
		require.ErrorIs(t, err, errTestDownstream)
	}
	require.Equal(t, Open, cb.state)

	// Test Open State
	_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errTestDownstream
	})
	require.ErrorIs(t, err, ErrTripped)
	require.Equal(t, Open, cb.state)

	// Test Half-Open State
	c.Tick(2 * time.Minute) // Advance time to reset
	_, err = cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 42, nil
	})
	require.NoError(t, err)
	require.Equal(t, HalfOpen, cb.state)
}

func TestCircuitBreakerFailureRatio(t *testing.T) {
	run := func(cb *CB[int], fail bool) {
		_, _ = cb.Do(context.Background(), func(ctx context.Context) (int, error) {
			if fail {
				return 0, errTestDownstream
			}
			return 0, nil
		})
	}

	t.Run("low failure rate does not trip even with many absolute failures", func(t *testing.T) {
		c := clock.NewTestClock()
		// 50% ratio, but a high-throughput window: 1000 ok + 5 fail = 0.5% << 50%.
		cb := New[int]("test", WithClock(c), WithCyclicPeriod(time.Hour),
			WithFailureRatio(0.5, 20))
		for i := 0; i < 1000; i++ {
			run(cb, false)
		}
		for i := 0; i < 5; i++ {
			run(cb, true)
		}
		require.Equal(t, Closed, cb.state, "0.5%% failure rate must not trip a 50%% breaker")
	})

	t.Run("high failure rate trips once past minRequests", func(t *testing.T) {
		c := clock.NewTestClock()
		cb := New[int]("test", WithClock(c), WithCyclicPeriod(time.Hour),
			WithFailureRatio(0.5, 20))
		// Below minRequests: all failing but sample too small to act.
		for i := 0; i < 19; i++ {
			run(cb, true)
		}
		require.Equal(t, Closed, cb.state, "must not trip before minRequests")
		run(cb, true) // 20th failing request crosses minRequests at 100% failure
		require.Equal(t, Open, cb.state, "100%% failure past minRequests must trip")
	})

	t.Run("ratio above 1 is clamped to 1 and still trips at 100% failure", func(t *testing.T) {
		c := clock.NewTestClock()
		// A typo'd ratio of 1.5 must not silently produce a never-tripping breaker.
		cb := New[int]("test", WithClock(c), WithCyclicPeriod(time.Hour),
			WithFailureRatio(1.5, 5))
		require.Equal(t, 1.0, cb.config.failureRatio, "ratio above 1 must clamp to 1")
		for i := 0; i < 5; i++ {
			run(cb, true)
		}
		require.Equal(t, Open, cb.state, "100%% failure past minRequests must trip a clamped breaker")
	})
}

func TestCircuitBreakerReset(t *testing.T) {

	c := clock.NewTestClock()
	cb := New[int]("test", WithCyclicPeriod(5*time.Second), WithClock(c), WithTripThreshold(3), WithTimeout(20*time.Second))

	// Trigger circuit breaker to open
	for i := 0; i < 3; i++ {
		_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
			return 0, errTestDownstream
		})
		require.ErrorIs(t, err, errTestDownstream)
	}

	require.Equal(t, Open, cb.state)

	// Advance time to reset
	c.Tick(30 * time.Second)

	// Next request should be allowed (Half-Open state)
	_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 42, nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	require.Equal(t, HalfOpen, cb.state)

}

func TestCircuitBreakerRecovers(t *testing.T) {

	cb := New[int]("test", WithMaxRequests(2))

	// Reset to Half-Open state
	cb.state = HalfOpen

	// Two requests should succeed
	for i := 0; i < 2; i++ {
		_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
			return 42, nil
		})
		require.NoError(t, err)
	}

	// Circuit should close
	require.Equal(t, Closed, cb.state)
}

func TestCircuitBreakerIgnoresContextCanceled(t *testing.T) {
	t.Parallel()

	cb := New[int]("test", WithTripThreshold(1))

	_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, context.Canceled
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, Closed, cb.state)
	require.Zero(t, cb.failures)
	require.Zero(t, cb.successes)

	_, err = cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errTestDownstream
	})
	require.ErrorIs(t, err, errTestDownstream)
	require.Equal(t, Open, cb.state)
}

func TestCircuitBreakerIgnoredErrorsDoNotCloseHalfOpenCircuit(t *testing.T) {
	t.Parallel()

	cb := New[int](
		"test",
		WithMaxRequests(1),
		WithIsDownstreamError(func(err error) bool {
			return err != nil && !errors.Is(err, context.Canceled)
		}),
	)
	cb.state = HalfOpen

	_, err := cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, context.Canceled
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, HalfOpen, cb.state)
	require.Zero(t, cb.consecutiveSuccesses)
	require.Zero(t, cb.consecutiveFailures)

	_, err = cb.Do(context.Background(), func(ctx context.Context) (int, error) {
		return 42, nil
	})
	require.NoError(t, err)
	require.Equal(t, Closed, cb.state)
}
