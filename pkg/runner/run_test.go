package runner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRun_ExecutesTasks verifies that Run starts all registered tasks
// concurrently when called.
func TestRun_ExecutesTasks(t *testing.T) {
	r := New()

	var executed atomic.Int32

	r.Go(func(ctx context.Context) error {
		executed.Add(1)
		<-ctx.Done()
		return nil
	})

	r.Go(func(ctx context.Context) error {
		executed.Add(1)
		<-ctx.Done()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- r.Wait(ctx, WithTimeout(100*time.Millisecond))
	}()

	require.Eventually(t, func() bool {
		return executed.Load() == 2
	}, time.Second, 10*time.Millisecond, "both tasks should start")

	cancel()

	err := <-done
	require.NoError(t, err)
}

// TestRun_StopsOnContextCancel verifies that canceling the parent context
// signals all tasks to stop via their context.
func TestRun_StopsOnContextCancel(t *testing.T) {
	r := New()

	taskStopped := make(chan struct{})
	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		close(taskStopped)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- r.Wait(ctx, WithTimeout(100*time.Millisecond))
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-taskStopped:
	case <-time.After(time.Second):
		t.Fatal("task should stop after context cancel")
	}

	err := <-done
	require.NoError(t, err)
}

// TestRun_StopsOnTaskError verifies that when any task returns a non-nil
// error, all other tasks are signaled to stop and the error is returned.
func TestRun_StopsOnTaskError(t *testing.T) {
	r := New()

	taskErr := errors.New("task failed")
	r.Go(func(ctx context.Context) error {
		return taskErr
	})

	otherTaskStopped := make(chan struct{})
	r.Go(func(ctx context.Context) error {
		<-ctx.Done()
		close(otherTaskStopped)
		return nil
	})

	err := r.Wait(context.Background(), WithTimeout(time.Second))

	require.ErrorIs(t, err, taskErr)

	select {
	case <-otherTaskStopped:
	case <-time.After(time.Second):
		t.Fatal("other task should stop after first task error")
	}
}

// TestRun_CanOnlyRunOnce verifies that Run returns an error if called while
// already running. This prevents undefined behavior from concurrent Run calls.
func TestRun_CanOnlyRunOnce(t *testing.T) {
	r := New()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- r.Wait(ctx, WithTimeout(100*time.Millisecond))
	}()

	time.Sleep(20 * time.Millisecond)

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.Error(t, err)
	require.Contains(t, err.Error(), "already running")

	cancel()
	<-done
}

// TestRun_EmptyRunner verifies that a runner with no tasks or cleanups exits
// cleanly when the context is canceled.
func TestRun_EmptyRunner(t *testing.T) {
	r := New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
}

// TestRun_CleanupsExecutedInReverseOrder verifies that cleanup handlers run
// in reverse registration order (LIFO). This matches defer semantics and
// ensures resources are released in the correct order.
func TestRun_CleanupsExecutedInReverseOrder(t *testing.T) {
	r := New()

	var order []int
	var mu sync.Mutex

	r.Defer(func() error {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		return nil
	})

	r.Defer(func() error {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		return nil
	})

	r.Defer(func() error {
		mu.Lock()
		order = append(order, 3)
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	require.Equal(t, []int{3, 2, 1}, order)
}

// TestRun_CleanupsReceiveContext verifies that context-aware cleanup handlers
// receive a valid context with a deadline for timeout enforcement.
func TestRun_CleanupsReceiveContext(t *testing.T) {
	r := New()

	var receivedCtx context.Context
	r.DeferCtx(func(ctx context.Context) error {
		receivedCtx = ctx
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(500*time.Millisecond))
	require.NoError(t, err)
	require.NotNil(t, receivedCtx)
}

// TestRun_ContinuesOnCleanupError verifies that all cleanup handlers run even
// if one returns an error. This ensures resources are properly released even
// during partial failures.
func TestRun_ContinuesOnCleanupError(t *testing.T) {
	r := New()

	var called []int
	var mu sync.Mutex

	r.Defer(func() error {
		mu.Lock()
		called = append(called, 1)
		mu.Unlock()
		return nil
	})

	r.Defer(func() error {
		mu.Lock()
		called = append(called, 2)
		mu.Unlock()
		return errors.New("cleanup 2 failed")
	})

	r.Defer(func() error {
		mu.Lock()
		called = append(called, 3)
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(100*time.Millisecond))
	require.Error(t, err)

	mu.Lock()
	require.Equal(t, []int{3, 2, 1}, called, "all cleanups should run even after error")
	mu.Unlock()
}

// TestWithTimeout_SetsTimeout verifies that WithTimeout sets a deadline on
// the cleanup context. Cleanup handlers that exceed this timeout will have
// their context canceled.
func TestWithTimeout_SetsTimeout(t *testing.T) {
	r := New()

	var cleanupCtx context.Context
	r.DeferCtx(func(ctx context.Context) error {
		cleanupCtx = ctx
		time.Sleep(200 * time.Millisecond)
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Wait(ctx, WithTimeout(50*time.Millisecond))
	require.ErrorIs(t, err, context.DeadlineExceeded)

	deadline, ok := cleanupCtx.Deadline()
	require.True(t, ok, "cleanup context should have deadline")
	require.False(t, deadline.IsZero())
}
