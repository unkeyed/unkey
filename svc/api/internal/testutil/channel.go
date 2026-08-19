package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Receive waits for one value from channel and fails the test if the channel
// closes or the timeout expires first.
func Receive[T any](t testing.TB, channel <-chan T, timeout time.Duration) T {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case value, open := <-channel:
		require.True(t, open, "channel closed before sending a value")
		return value
	case <-timer.C:
		require.FailNow(t, "timed out waiting for channel value", "timeout: %s", timeout)
		var zero T
		return zero
	}
}

// RequireNoReceive fails the test if channel receives a value before the
// timeout. Use it to verify that asynchronous work was not submitted.
func RequireNoReceive[T any](t testing.TB, channel <-chan T, timeout time.Duration) {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case value, open := <-channel:
		if !open {
			require.FailNow(t, "channel closed unexpectedly")
		}
		require.FailNow(t, "received unexpected channel value", "value: %+v", value)
	case <-timer.C:
	}
}
