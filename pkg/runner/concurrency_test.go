package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// TestConcurrency_GoIgnoredDuringShutdown verifies that tasks cannot be
// registered after shutdown has started.
func TestConcurrency_GoIgnoredDuringShutdown(t *testing.T) {
	r := New()

	var taskRan atomic.Bool

	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		done <- r.Wait(ctx, WithTimeout(100*time.Millisecond))
	}()

	<-started
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for shutdown to start
	time.Sleep(10 * time.Millisecond)

	// Try to register a task during shutdown - should be ignored
	r.Go(func(ctx context.Context) error {
		taskRan.Store(true)
		return nil
	})

	<-done

	require.False(t, taskRan.Load(), "task registered during shutdown should not run")
}

// TestConcurrency_Registration verifies that Go and Defer are safe for
// concurrent use from multiple goroutines.
func TestConcurrency_Registration(t *testing.T) {
	r := New(logging.NewNoop())

	const numGoroutines = 10
	const itemsPerGoroutine = 50

	var taskCount atomic.Int32
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
					taskCount.Add(1)
					<-ctx.Done()
					return nil
				})
			}
		}()
	}

	wg.Wait()

	// Give goroutines time to increment counter
	time.Sleep(50 * time.Millisecond)

	r.mu.Lock()
	cleanupCount := len(r.cleanups)
	r.mu.Unlock()

	require.Equal(t, numGoroutines*itemsPerGoroutine, cleanupCount)
	require.Equal(t, int32(numGoroutines*itemsPerGoroutine), taskCount.Load())

	// Cleanup
	r.cancel()
	r.wg.Wait()
}

// TestConcurrency_DeferIgnoredDuringShutdown verifies that Defer calls made
// while shutdown is in progress are silently ignored.
func TestConcurrency_DeferIgnoredDuringShutdown(t *testing.T) {
	r := New(logging.NewNoop())

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
		_ = r.Wait(ctx, WithTimeout(time.Second))
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

// TestConcurrency_GoStartsImmediately verifies that Go() starts tasks immediately,
// not when Run() is called.
func TestConcurrency_GoStartsImmediately(t *testing.T) {
	r := New()

	started := make(chan struct{})
	r.Go(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return nil
	})

	select {
	case <-started:
		// Task started immediately - good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("task did not start immediately")
	}

	// Cleanup
	r.cancel()
	r.wg.Wait()
}
