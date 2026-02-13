package repeat

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvery_BasicFunctionality(t *testing.T) {
	t.Run("calls function repeatedly", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, nil, func() {
			counter.Add(1)
		})

		// Wait for several calls
		time.Sleep(55 * time.Millisecond)
		stop()

		count := counter.Load()
		assert.GreaterOrEqual(t, count, int32(3), "should have called function at least 3 times")
		assert.LessOrEqual(t, count, int32(8), "should not have called function too many times")
	})

	t.Run("stops when stop function is called", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, nil, func() {
			counter.Add(1)
		})

		// Let it run briefly
		time.Sleep(25 * time.Millisecond)
		stop()

		// Get count after stopping
		countAfterStop := counter.Load()

		// Wait more time to ensure it really stopped
		time.Sleep(30 * time.Millisecond)
		finalCount := counter.Load()

		assert.Equal(t, countAfterStop, finalCount, "function should not be called after stop")
	})
}

func TestEvery_GoroutineLeak(t *testing.T) {
	t.Run("no goroutine leak with normal stop", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		var stops []func()
		for range 10 {
			stop := Every(100*time.Millisecond, nil, func() {
				// Do minimal work
			})
			stops = append(stops, stop)
		}

		// Stop all
		for _, stop := range stops {
			stop()
		}

		// Wait for cleanup
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.GC() // Double GC to be thorough

		finalGoroutines := runtime.NumGoroutine()

		// Allow some tolerance for test framework goroutines
		assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2,
			"should not have significant goroutine leak")
	})

	t.Run("immediate stop does not leak", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		for range 5 {
			stop := Every(1*time.Millisecond, nil, func() {})
			stop() // Stop immediately
		}

		time.Sleep(50 * time.Millisecond)
		runtime.GC()

		finalGoroutines := runtime.NumGoroutine()
		assert.LessOrEqual(t, finalGoroutines, initialGoroutines+1,
			"immediate stop should not leak goroutines")
	})
}

func TestEvery_PanicRecovery(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		var callCount atomic.Int32

		stop := Every(5*time.Millisecond, nil, func() {
			count := callCount.Add(1)
			if count == 2 {
				panic("test panic")
			}
		})

		// Wait longer for panic to occur and recovery
		time.Sleep(100 * time.Millisecond)
		stop()

		// Should have continued calling after panic
		assert.GreaterOrEqual(t, callCount.Load(), int32(3),
			"should continue calling function after panic")
	})

	t.Run("panic in function does not crash program", func(t *testing.T) {
		done := make(chan bool)

		stop := Every(5*time.Millisecond, nil, func() {
			panic("intentional panic")
		})

		// Run for a bit to ensure multiple panics are handled
		go func() {
			time.Sleep(30 * time.Millisecond)
			stop()
			done <- true
		}()

		select {
		case <-done:
			// Success - program didn't crash
		case <-time.After(100 * time.Millisecond):
			t.Fatal("test timed out - program may have crashed")
		}
	})
}

func TestEvery_ConcurrentStops(t *testing.T) {
	t.Run("multiple stop calls are safe", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(5*time.Millisecond, nil, func() {
			counter.Add(1)
		})

		// Call stop multiple times sequentially
		stop()
		stop()
		stop()

		time.Sleep(20 * time.Millisecond)

		// Function should have stopped after first stop call
		// Counter should be low since we stopped quickly
		assert.LessOrEqual(t, counter.Load(), int32(3),
			"function should stop after first stop call")
	})

	t.Run("concurrent stop calls are safe", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(5*time.Millisecond, nil, func() {
			counter.Add(1)
		})

		// Call stop concurrently from multiple goroutines
		var wg sync.WaitGroup
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stop()
			}()
		}

		wg.Wait()
		time.Sleep(20 * time.Millisecond)

		// Should not panic or cause issues
		assert.LessOrEqual(t, counter.Load(), int32(5),
			"concurrent stops should work without issues")
	})
}

func TestEvery_EdgeCases(t *testing.T) {
	t.Run("very short interval", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(1*time.Millisecond, nil, func() {
			counter.Add(1)
		})

		time.Sleep(50 * time.Millisecond)
		stop()

		// Should handle very short intervals without issues
		count := counter.Load()
		assert.Greater(t, count, int32(10), "should handle very short intervals")
		assert.Less(t, count, int32(100), "should not be unreasonably high")
	})

	t.Run("very long interval", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(1*time.Hour, nil, func() {
			counter.Add(1)
		})

		// Should handle very long intervals without issues
		time.Sleep(10 * time.Millisecond)
		stop()

		// Should have been called once
		count := counter.Load()
		assert.Equal(t, int32(1), count, "should have been called once")
	})

	t.Run("function that takes longer than interval", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, nil, func() {
			counter.Add(1)
			time.Sleep(20 * time.Millisecond) // Longer than interval
		})

		time.Sleep(60 * time.Millisecond)
		stop()

		// Should handle slow functions gracefully
		count := counter.Load()
		assert.GreaterOrEqual(t, count, int32(1), "should call slow function at least once")
		assert.LessOrEqual(t, count, int32(5), "should not stack up too many calls")
	})
}

func TestEvery_StopBehavior(t *testing.T) {
	t.Run("stop is idempotent", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, nil, func() {
			counter.Add(1)
		})

		time.Sleep(25 * time.Millisecond)

		// First stop
		stop()
		countAfterFirstStop := counter.Load()

		time.Sleep(20 * time.Millisecond)

		// Second stop should be safe
		stop()
		countAfterSecondStop := counter.Load()

		assert.Equal(t, countAfterFirstStop, countAfterSecondStop,
			"second stop call should have no effect")
	})

	t.Run("stop returns immediately", func(t *testing.T) {

		stop := Every(100*time.Millisecond, nil, func() {
			time.Sleep(50 * time.Millisecond)
		})

		start := time.Now()
		stop()
		elapsed := time.Since(start)

		assert.Less(t, elapsed, 50*time.Millisecond,
			"stop should return quickly even if function is running")
	})
}

// Benchmark tests to measure performance impact
func BenchmarkEvery(b *testing.B) {

	b.Run("StartStop", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			stop := Every(100*time.Millisecond, nil, func() {})
			stop()
		}
	})

	b.Run("WithFunction", func(b *testing.B) {
		var counter atomic.Int64

		stop := Every(1*time.Microsecond, nil, func() {
			counter.Add(1)
		})

		b.ResetTimer()
		for b.Loop() {
			// Just let it run
			runtime.Gosched()
		}

		stop()
	})
}
