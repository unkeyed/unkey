package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
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
