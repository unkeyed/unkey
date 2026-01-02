package shutdown

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	shutdowns := New()
	require.NotNil(t, shutdowns)
	require.Empty(t, shutdowns.functions)
}

func TestRegister(t *testing.T) {
	shutdowns := New()

	// Register a simple function
	called := false
	fn := func() error {
		called = true
		return nil
	}

	shutdowns.Register(fn)

	shutdowns.mu.RLock()
	require.Len(t, shutdowns.functions, 1)
	shutdowns.mu.RUnlock()

	// Call the registered function to verify it works
	shutdowns.mu.RLock()
	err := shutdowns.functions[0](context.Background())
	shutdowns.mu.RUnlock()

	require.NoError(t, err)
	require.True(t, called)
}

func TestRegisterMultiple(t *testing.T) {
	shutdowns := New()

	// Register multiple simple functions at once
	called1 := false
	called2 := false
	called3 := false

	fn1 := func() error {
		called1 = true
		return nil
	}

	fn2 := func() error {
		called2 = true
		return nil
	}

	fn3 := func() error {
		called3 = true
		return nil
	}

	// Register all functions at once
	shutdowns.Register(fn1, fn2, fn3)

	shutdowns.mu.RLock()
	require.Len(t, shutdowns.functions, 3)
	shutdowns.mu.RUnlock()

	// Call the registered functions to verify they work
	shutdowns.mu.RLock()
	err := shutdowns.functions[0](context.Background())
	shutdowns.mu.RUnlock()
	require.NoError(t, err)
	require.True(t, called1)

	shutdowns.mu.RLock()
	err = shutdowns.functions[1](context.Background())
	shutdowns.mu.RUnlock()
	require.NoError(t, err)
	require.True(t, called2)

	shutdowns.mu.RLock()
	err = shutdowns.functions[2](context.Background())
	shutdowns.mu.RUnlock()
	require.NoError(t, err)
	require.True(t, called3)
}

func TestRegisterCtx(t *testing.T) {
	shutdowns := New()

	// Register a context-aware function
	called := false
	fn := func(ctx context.Context) error {
		called = true
		return nil
	}

	shutdowns.RegisterCtx(fn)

	shutdowns.mu.RLock()
	require.Len(t, shutdowns.functions, 1)
	shutdowns.mu.RUnlock()

	// Call the registered function to verify it works
	shutdowns.mu.RLock()
	err := shutdowns.functions[0](context.Background())
	shutdowns.mu.RUnlock()

	require.NoError(t, err)
	require.True(t, called)
}

func TestRegisterCtxMultiple(t *testing.T) {
	shutdowns := New()

	// Register multiple context-aware functions at once
	called1 := false
	called2 := false
	called3 := false

	fn1 := func(ctx context.Context) error {
		called1 = true
		return nil
	}

	fn2 := func(ctx context.Context) error {
		called2 = true
		return nil
	}

	fn3 := func(ctx context.Context) error {
		called3 = true
		return nil
	}

	// Register all functions at once
	shutdowns.RegisterCtx(fn1, fn2, fn3)

	shutdowns.mu.RLock()
	require.Len(t, shutdowns.functions, 3)
	shutdowns.mu.RUnlock()

	// Call the registered functions to verify they work
	shutdowns.mu.RLock()
	err := shutdowns.functions[0](context.Background())
	shutdowns.mu.RUnlock()
	require.NoError(t, err)
	require.True(t, called1)

	shutdowns.mu.RLock()
	err = shutdowns.functions[1](context.Background())
	shutdowns.mu.RUnlock()
	require.NoError(t, err)
	require.True(t, called2)

	shutdowns.mu.RLock()
	err = shutdowns.functions[2](context.Background())
	shutdowns.mu.RUnlock()
	require.NoError(t, err)
	require.True(t, called3)
}

func TestShutdown(t *testing.T) {
	tests := []struct {
		name      string
		functions []Shutdown
		wantErr   bool
	}{
		{
			name:      "empty shutdowns",
			functions: []Shutdown{},
			wantErr:   false,
		},
		{
			name: "successful shutdowns",
			functions: []Shutdown{
				func() error { return nil },
				func() error { return nil },
			},
			wantErr: false,
		},
		{
			name: "one error",
			functions: []Shutdown{
				func() error { return nil },
				func() error { return errors.New("shutdown error") },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdowns := New()

			// Register all test functions
			shutdowns.Register(tt.functions...)

			// Execute shutdown
			errs := shutdowns.Shutdown(context.Background())

			if tt.wantErr {
				require.Greater(t, len(errs), 0)
			} else {
				require.Empty(t, errs)
			}
		})
	}
}

func TestShutdownOrder(t *testing.T) {
	shutdowns := New()

	// Track order of execution
	var order []int
	mutex := make(chan struct{}, 1)
	mutex <- struct{}{}

	// Register functions in order 1, 2, 3
	shutdowns.Register(
		func() error {
			<-mutex
			order = append(order, 1)
			mutex <- struct{}{}
			return nil
		},
		func() error {
			<-mutex
			order = append(order, 2)
			mutex <- struct{}{}
			return nil
		},
		func() error {
			<-mutex
			order = append(order, 3)
			mutex <- struct{}{}
			return nil
		},
	)

	// Shutdown should execute in reverse order: 3, 2, 1
	errs := shutdowns.Shutdown(context.Background())
	require.Empty(t, errs)
	require.Equal(t, []int{3, 2, 1}, order)
}

func TestShutdownOrderWithMultipleRegistrations(t *testing.T) {
	shutdowns := New()

	// Track order of execution
	var order []int
	mutex := make(chan struct{}, 1)
	mutex <- struct{}{}

	// Register functions in batches
	shutdowns.Register(
		func() error {
			<-mutex
			order = append(order, 1)
			mutex <- struct{}{}
			return nil
		},
		func() error {
			<-mutex
			order = append(order, 2)
			mutex <- struct{}{}
			return nil
		},
	)

	shutdowns.RegisterCtx(
		func(ctx context.Context) error {
			<-mutex
			order = append(order, 3)
			mutex <- struct{}{}
			return nil
		},
		func(ctx context.Context) error {
			<-mutex
			order = append(order, 4)
			mutex <- struct{}{}
			return nil
		},
	)

	// Shutdown should execute in reverse order: 4, 3, 2, 1
	errs := shutdowns.Shutdown(context.Background())
	require.Empty(t, errs)
	require.Equal(t, []int{4, 3, 2, 1}, order)
}

func TestShutdownContext(t *testing.T) {
	shutdowns := New()

	// Register a function that respects context cancellation
	shutdowns.RegisterCtx(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return nil
		}
	})

	// Test with a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Shutdown should return context.Canceled
	errs := shutdowns.Shutdown(ctx)
	require.Len(t, errs, 1)
	require.ErrorIs(t, errs[0], context.Canceled)

	// Test with a context timeout
	shutdowns = New()
	shutdowns.RegisterCtx(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Shutdown should succeed
	errs = shutdowns.Shutdown(ctx)
	require.Empty(t, errs)
}

func TestShutdownErrorHandling(t *testing.T) {
	shutdowns := New()

	// If one function fails, others after it should not be called
	firstCalled := false
	secondCalled := false
	thirdCalled := false

	shutdowns.Register(func() error {
		thirdCalled = true
		return nil
	})

	shutdowns.Register(func() error {
		secondCalled = true
		return errors.New("second function failed")
	})

	shutdowns.Register(func() error {
		firstCalled = true
		return nil
	})

	errs := shutdowns.Shutdown(context.Background())
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "second function failed")

	require.True(t, firstCalled, "First function should be called")
	require.True(t, secondCalled, "Second function should be called (and fail)")
	require.True(t, thirdCalled, "Third function should be called after error")
}

// New concurrency tests

func TestConcurrentRegistration(t *testing.T) {
	shutdowns := New()
	const numGoroutines = 10
	const funcsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently register many functions
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < funcsPerGoroutine; j++ {
				shutdowns.Register(func() error {
					return nil
				})
			}
		}()
	}

	wg.Wait()

	// Verify all functions were registered
	shutdowns.mu.RLock()
	numFunctions := len(shutdowns.functions)
	shutdowns.mu.RUnlock()

	require.Equal(t, numGoroutines*funcsPerGoroutine, numFunctions,
		"Expected all functions to be registered")
}

func TestConcurrentRegistrationAndShutdown(t *testing.T) {
	shutdowns := New()

	// Channel to synchronize the test
	start := make(chan struct{})
	var registrationDone sync.WaitGroup

	// Counter for registered functions
	counter := 0
	var counterMu sync.Mutex

	// Launch goroutines that will register functions
	const numRegistrationGoroutines = 5
	registrationDone.Add(numRegistrationGoroutines)

	for i := 0; i < numRegistrationGoroutines; i++ {
		go func() {
			defer registrationDone.Done()

			<-start // Wait for the signal to start

			// Try to register some functions while shutdown might be happening
			for j := 0; j < 20; j++ {
				shutdowns.Register(func() error {
					counterMu.Lock()
					counter++
					counterMu.Unlock()
					return nil
				})

				// Small randomization to increase chance of race conditions
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Launch shutdown in its own goroutine
	var shutdownErrs []error
	var shutdownDone sync.WaitGroup
	shutdownDone.Add(1)

	go func() {
		defer shutdownDone.Done()

		// Give registration goroutines a small head start
		close(start)
		time.Sleep(10 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		shutdownErrs = shutdowns.Shutdown(ctx)
	}()

	// Wait for registration goroutines to finish their attempts
	registrationDone.Wait()
	shutdownDone.Wait()

	// Check if shutdown completed successfully
	require.Empty(t, shutdownErrs, "Shutdown should complete without errors")

	// During shutdown, we should have set shuttingDown=true
	require.True(t, shutdowns.shuttingDown, "shuttingDown flag should be set")

	// There are two valid outcomes:
	// 1. All registrations happened before shutdown started
	// 2. Some registrations were ignored because they happened during/after shutdown
	// Either way, registered functions should have executed correctly
	counterMu.Lock()
	execCount := counter
	counterMu.Unlock()

	t.Logf("Executed %d shutdown functions", execCount)
	// The number of executed functions should be less than or equal to
	// the total possible registrations
	require.LessOrEqual(t, execCount, numRegistrationGoroutines*20)
}

func TestRegisterNilFunctions(t *testing.T) {
	shutdowns := New()

	// These should not panic
	shutdowns.Register()
	shutdowns.RegisterCtx()

	shutdowns.mu.RLock()
	count := len(shutdowns.functions)
	shutdowns.mu.RUnlock()

	require.Equal(t, 0, count, "No functions should be registered when passing empty slices")
}

func TestRegisterDuringShutdown(t *testing.T) {
	shutdowns := New()

	// This test verifies that registrations during shutdown are ignored
	var order []int
	var orderMu sync.Mutex

	// Add a long-running shutdown function
	shutdowns.RegisterCtx(func(ctx context.Context) error {
		orderMu.Lock()
		order = append(order, 1)
		orderMu.Unlock()

		// Sleep to give time for the registration to happen
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Start shutdown in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		shutdowns.Shutdown(context.Background())
	}()

	// Give shutdown a moment to start
	time.Sleep(20 * time.Millisecond)

	// Try to register a function during shutdown
	shutdowns.Register(func() error {
		orderMu.Lock()
		order = append(order, 2)
		orderMu.Unlock()
		return nil
	})

	wg.Wait()

	// The second function should never be called
	require.Equal(t, []int{1}, order, "Only the first function should be executed")
}
