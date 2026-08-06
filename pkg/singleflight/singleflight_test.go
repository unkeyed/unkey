package singleflight

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	group := New[string, string]()
	value, err := group.Do(context.Background(), "key", func(context.Context) (string, error) {
		return "bar", nil
	})
	require.NoError(t, err)
	require.Equal(t, "bar", value)
}

func TestDoReturnsError(t *testing.T) {
	group := New[string, *struct{}]()
	expectedErr := errors.New("some error")
	value, err := group.Do(context.Background(), "key", func(context.Context) (*struct{}, error) {
		return nil, expectedErr
	})
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, value)
}

// TestDoDeduplicatesConcurrentCalls guarantees that all callers already waiting
// for the same key receive one execution's result.
func TestDoDeduplicatesConcurrentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := New[string, string]()
		releaseCall := make(chan struct{})
		var callCount atomic.Int32
		function := func(context.Context) (string, error) {
			callCount.Add(1)
			<-releaseCall
			return "bar", nil
		}

		const callerCount = 10
		callErrors := make(chan error, callerCount)
		callValues := make(chan string, callerCount)
		for range callerCount {
			go func() {
				value, err := group.Do(context.Background(), "key", function)
				callErrors <- err
				callValues <- value
			}()
		}

		synctest.Wait()
		require.Equal(t, int32(1), callCount.Load())
		close(releaseCall)
		synctest.Wait()

		for range callerCount {
			require.NoError(t, <-callErrors)
			require.Equal(t, "bar", <-callValues)
		}
	})
}

// TestDoAsyncReservesBeforeEnqueue guarantees that queued work remains
// deduplicated before a worker begins executing it.
func TestDoAsyncReservesBeforeEnqueue(t *testing.T) {
	group := New[string, string]()
	var scheduled func()
	schedule := func(function func()) {
		scheduled = function
	}

	require.True(t, group.DoAsync(context.Background(), "key", schedule, func(context.Context) (string, error) {
		return "value", nil
	}))
	require.False(t, group.DoAsync(context.Background(), "key", schedule, func(context.Context) (string, error) {
		return "duplicate", nil
	}))
	require.NotNil(t, scheduled)
	scheduled()

	value, err := group.Do(context.Background(), "key", func(context.Context) (string, error) {
		return "next", nil
	})
	require.NoError(t, err)
	require.Equal(t, "next", value)
}

func TestDoAsyncPanicReleasesKey(t *testing.T) {
	group := New[string, string]()
	require.Panics(t, func() {
		group.DoAsync(context.Background(), "key", func(func()) {
			panic("scheduler failed")
		}, func(context.Context) (string, error) {
			return "unreachable", nil
		})
	})

	value, err := group.Do(context.Background(), "key", func(context.Context) (string, error) {
		return "recovered", nil
	})
	require.NoError(t, err)
	require.Equal(t, "recovered", value)
}

// TestCanceledWaiterStopsWaiting guarantees that one waiting caller can cancel
// without interrupting the call shared by other waiters.
func TestCanceledWaiterStopsWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := New[string, string]()
		releaseCall := make(chan struct{})
		executingDone := make(chan error, 1)
		go func() {
			_, err := group.Do(context.Background(), "key", func(context.Context) (string, error) {
				<-releaseCall
				return "value", nil
			})
			executingDone <- err
		}()
		synctest.Wait()

		type callResult struct {
			value string
			err   error
		}
		waitingCtx, cancelWaiting := context.WithCancel(context.Background())
		waitingDone := make(chan callResult, 1)
		go func() {
			value, err := group.Do(waitingCtx, "key", func(context.Context) (string, error) {
				return "unexpected", errors.New("waiting caller executed")
			})
			waitingDone <- callResult{value: value, err: err}
		}()
		synctest.Wait()

		cancelWaiting()
		synctest.Wait()
		waitingResult := <-waitingDone
		require.ErrorIs(t, waitingResult.err, context.Canceled)
		require.Empty(t, waitingResult.value)

		close(releaseCall)
		synctest.Wait()
		require.NoError(t, <-executingDone)
	})
}

func TestCanceledCallerDoesNotExecute(t *testing.T) {
	group := New[string, string]()
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	value, err := group.Do(canceledCtx, "key", func(context.Context) (string, error) {
		t.Fatal("a canceled caller must not execute the function")
		return "", nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, value)
}

// TestCanceledExecutingCallerDoesNotCancelWaiter guarantees that a healthy
// waiter retries instead of inheriting another caller's cancellation.
func TestCanceledExecutingCallerDoesNotCancelWaiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := New[string, string]()
		executingCtx, cancelExecuting := context.WithCancel(context.Background())
		executingDone := make(chan error, 1)
		go func() {
			_, err := group.Do(executingCtx, "key", func(ctx context.Context) (string, error) {
				<-ctx.Done()
				return "", ctx.Err()
			})
			executingDone <- err
		}()
		synctest.Wait()

		type callResult struct {
			value string
			err   error
		}
		waitingDone := make(chan callResult, 1)
		go func() {
			value, err := group.Do(context.Background(), "key", func(context.Context) (string, error) {
				return "value", nil
			})
			waitingDone <- callResult{value: value, err: err}
		}()
		synctest.Wait()

		cancelExecuting()
		synctest.Wait()
		require.ErrorIs(t, <-executingDone, context.Canceled)
		waitingResult := <-waitingDone
		require.NoError(t, waitingResult.err)
		require.Equal(t, "value", waitingResult.value)
	})
}

// TestPanicReleasesKey guarantees that a panicking call cannot leave future
// callers permanently blocked on the key.
func TestPanicReleasesKey(t *testing.T) {
	group := New[string, string]()

	func() {
		defer func() {
			require.Equal(t, "boom", recover())
		}()
		_, _ = group.Do(context.Background(), "key", func(context.Context) (string, error) {
			panic("boom")
		})
	}()

	value, err := group.Do(context.Background(), "key", func(context.Context) (string, error) {
		return "recovered", nil
	})
	require.NoError(t, err)
	require.Equal(t, "recovered", value)
}
