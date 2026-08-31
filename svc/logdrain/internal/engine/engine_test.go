package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestBackoff guarantees the retry schedule and the total wait before the
// default failure threshold.
func TestBackoff(t *testing.T) {
	want := []time.Duration{
		1 * time.Minute,
		2 * time.Minute,
		4 * time.Minute,
		8 * time.Minute,
		16 * time.Minute,
		32 * time.Minute,
		64 * time.Minute,
		128 * time.Minute,
		4 * time.Hour,
		4 * time.Hour,
	}
	for failures, delay := range want {
		require.Equal(t, delay, backoff(failures))
	}

	var total time.Duration
	for failures := range 49 {
		total += backoff(failures)
	}
	require.Equal(t, 7*24*time.Hour+15*time.Minute, total)
}

// TestRetryDelay guarantees destination hints can extend local backoff but
// cannot exceed the one-day cap.
func TestRetryDelay(t *testing.T) {
	require.Equal(t, time.Minute, retryDelay(0, 30*time.Second))
	require.Equal(t, 2*time.Hour, retryDelay(0, 2*time.Hour))
	require.Equal(t, 24*time.Hour, retryDelay(0, 7*24*time.Hour))
	require.Equal(t, 4*time.Hour, retryDelay(8, 0))
}
