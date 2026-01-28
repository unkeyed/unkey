package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestConcurrency_GoIgnoredAfterRunning verifies that tasks cannot be
// registered after Run has started. This prevents race conditions where late
// registrations would be silently ignored or cause undefined behavior.
func TestConcurrency_GoIgnoredAfterRunning(t *testing.T) {
	r := New()

	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		done <- r.Run(ctx, WithTimeout(100*time.Millisecond))
	}()

	<-started
	time.Sleep(10 * time.Millisecond)

	r.Go(func(ctx context.Context) error {
		t.Error("this task should not be registered")
		return nil
	})

	r.mu.Lock()
	taskCount := len(r.tasks)
	r.mu.Unlock()

	require.Equal(t, 1, taskCount, "should not add task after Run started")

	cancel()
	<-done
}

// TestConcurrency_Registration verifies that Go and Defer are safe for
// concurrent use from multiple goroutines. All registrations should succeed
// without data races.
func TestConcurrency_Registration(t *testing.T) {
	r := New()

	const numGoroutines = 10
	const itemsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				r.Defer(func() error { return nil })
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				r.Go(func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				})
			}
		}()
	}

	wg.Wait()

	r.mu.Lock()
	cleanupCount := len(r.cleanups)
	taskCount := len(r.tasks)
	r.mu.Unlock()

	require.Equal(t, numGoroutines*itemsPerGoroutine, cleanupCount)
	require.Equal(t, numGoroutines*itemsPerGoroutine, taskCount)
}

// TestConcurrency_DeferIgnoredDuringShutdown verifies that Defer calls made
// while shutdown is in progress are silently ignored. This prevents cleanup
// handlers from being registered after the shutdown sequence has begun.
func TestConcurrency_DeferIgnoredDuringShutdown(t *testing.T) {
	r := New()

	registered := make(chan struct{})
	r.DeferCtx(func(ctx context.Context) error {
		close(registered)
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		_ = r.Run(ctx, WithTimeout(time.Second))
		close(done)
	}()

	<-registered

	r.Defer(func() error {
		t.Error("this cleanup should not be registered during shutdown")
		return nil
	})

	<-done

	r.mu.Lock()
	cleanupCount := len(r.cleanups)
	r.mu.Unlock()

	require.Equal(t, 1, cleanupCount, "cleanup registered during shutdown should be ignored")
}
