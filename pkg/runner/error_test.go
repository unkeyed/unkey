package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestError_CleanupErrorReturned verifies that errors from cleanup handlers
// are returned from Run.
func TestError_CleanupErrorReturned(t *testing.T) {
	r := New()

	cleanupErr := errors.New("cleanup failed")
	r.Defer(func() error {
		return cleanupErr
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Run(ctx, WithTimeout(100*time.Millisecond))
	require.ErrorIs(t, err, cleanupErr)
}

// TestError_TaskAndCleanupErrorsJoined verifies that when both a task fails
// and a cleanup fails, both errors are joined and returned together.
func TestError_TaskAndCleanupErrorsJoined(t *testing.T) {
	r := New()

	taskErr := errors.New("task failed")
	cleanupErr := errors.New("cleanup failed")

	r.Go(func(ctx context.Context) error {
		return taskErr
	})

	r.Defer(func() error {
		return cleanupErr
	})

	err := r.Run(context.Background(), WithTimeout(time.Second))
	require.ErrorIs(t, err, taskErr)
	require.ErrorIs(t, err, cleanupErr)
}

// TestError_ContextCanceledIgnored verifies that context.Canceled errors from
// tasks are not reported as failures. This is expected behavior when shutdown
// is triggered by context cancellation or signals.
func TestError_ContextCanceledIgnored(t *testing.T) {
	r := New()

	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx, WithTimeout(100*time.Millisecond))
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	err := <-done
	require.NoError(t, err, "context.Canceled from tasks should not be reported as error")
}

// TestError_FirstTaskErrorWins verifies that when multiple tasks fail
// concurrently, the first error received triggers shutdown and is reported.
// Subsequent errors from other tasks are not captured.
func TestError_FirstTaskErrorWins(t *testing.T) {
	r := New()

	firstErr := errors.New("first error")
	secondErr := errors.New("second error")

	started := make(chan struct{})

	r.Go(func(ctx context.Context) error {
		<-started
		return firstErr
	})

	r.Go(func(ctx context.Context) error {
		<-started
		time.Sleep(50 * time.Millisecond)
		return secondErr
	})

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), WithTimeout(time.Second))
	}()

	time.Sleep(20 * time.Millisecond)
	close(started)

	err := <-done
	require.ErrorIs(t, err, firstErr)
}
