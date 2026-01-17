package repeat

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvery_BasicFunctionality(t *testing.T) {
	t.Run("calls function repeatedly", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, func() {
			counter.Add(1)
		})
		defer stop()

		require.Eventually(t, func() bool {
			return counter.Load() >= 3
		}, 200*time.Millisecond, 5*time.Millisecond, "should have called function at least 3 times")

		stop()
		count := counter.Load()
		assert.LessOrEqual(t, count, int32(25), "should not have called function too many times")
	})

	t.Run("stops when stop function is called", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, func() {
			counter.Add(1)
		})

		require.Eventually(t, func() bool {
			return counter.Load() >= 1
		}, 200*time.Millisecond, 5*time.Millisecond, "should have called function at least once")

		stop()
		countAfterStop := counter.Load()

		require.Never(t, func() bool {
			return counter.Load() > countAfterStop
		}, 50*time.Millisecond, 5*time.Millisecond, "function should not be called after stop")
	})
}

func TestEvery_GoroutineLeak(t *testing.T) {
	t.Run("no goroutine leak with normal stop", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		var stops []func()
		for range 10 {
			stop := Every(100*time.Millisecond, func() {
				// Do minimal work
			})
			stops = append(stops, stop)
		}

		// Stop all
		for _, stop := range stops {
			stop()
		}

		require.Eventually(t, func() bool {
			runtime.GC()
			return runtime.NumGoroutine() <= initialGoroutines+2
		}, 500*time.Millisecond, 20*time.Millisecond, "should not have significant goroutine leak")
	})

	t.Run("immediate stop does not leak", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		for range 5 {
			stop := Every(1*time.Millisecond, func() {})
			stop() // Stop immediately
		}

		require.Eventually(t, func() bool {
			runtime.GC()
			return runtime.NumGoroutine() <= initialGoroutines+1
		}, 200*time.Millisecond, 10*time.Millisecond, "immediate stop should not leak goroutines")
	})
}

func TestEvery_PanicRecovery(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		var callCount atomic.Int32

		stop := Every(5*time.Millisecond, func() {
			count := callCount.Add(1)
			if count == 2 {
				panic("test panic")
			}
		})
		defer stop()

		require.Eventually(t, func() bool {
			return callCount.Load() >= 3
		}, 500*time.Millisecond, 5*time.Millisecond, "should continue calling function after panic")

		stop()
	})

	t.Run("panic in function does not crash program", func(t *testing.T) {
		done := make(chan bool)

		stop := Every(5*time.Millisecond, func() {
			panic("intentional panic")
		})

		go func() {
			defer func() { done <- true }()
			require.Eventually(t, func() bool {
				return true
			}, 30*time.Millisecond, 5*time.Millisecond)
			stop()
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

		stop := Every(5*time.Millisecond, func() {
			counter.Add(1)
		})

		// Wait for at least one call
		require.Eventually(t, func() bool {
			return counter.Load() >= 1
		}, 200*time.Millisecond, 5*time.Millisecond, "should have been called at least once")

		// Call stop multiple times sequentially
		stop()
		stop()
		stop()

		// Wait briefly for any in-flight execution to complete, then verify stability
		require.Eventually(t, func() bool {
			countBefore := counter.Load()
			time.Sleep(10 * time.Millisecond)
			return counter.Load() == countBefore
		}, 100*time.Millisecond, 5*time.Millisecond, "function should stop after first stop call")
	})

	t.Run("concurrent stop calls are safe", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(5*time.Millisecond, func() {
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
		countAfterStop := counter.Load()

		require.Never(t, func() bool {
			return counter.Load() > countAfterStop
		}, 50*time.Millisecond, 5*time.Millisecond, "concurrent stops should work without issues")
	})
}

func TestEvery_EdgeCases(t *testing.T) {
	t.Run("very short interval", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(1*time.Millisecond, func() {
			counter.Add(1)
		})
		defer stop()

		require.Eventually(t, func() bool {
			return counter.Load() >= 10
		}, 200*time.Millisecond, 5*time.Millisecond, "should handle very short intervals")

		stop()
		count := counter.Load()
		assert.Less(t, count, int32(500), "should not be unreasonably high")
	})

	t.Run("very long interval", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(1*time.Hour, func() {
			counter.Add(1)
		})
		defer stop()

		require.Eventually(t, func() bool {
			return counter.Load() >= 1
		}, 100*time.Millisecond, 5*time.Millisecond, "should have been called once")

		stop()
		count := counter.Load()
		assert.Equal(t, int32(1), count, "should have been called once")
	})

	t.Run("function that takes longer than interval", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, func() {
			counter.Add(1)
			time.Sleep(20 * time.Millisecond) // Longer than interval
		})
		defer stop()

		require.Eventually(t, func() bool {
			return counter.Load() >= 1
		}, 200*time.Millisecond, 5*time.Millisecond, "should call slow function at least once")

		stop()
		count := counter.Load()
		assert.LessOrEqual(t, count, int32(10), "should not stack up too many calls")
	})
}

func TestEvery_StopBehavior(t *testing.T) {
	t.Run("stop is idempotent", func(t *testing.T) {
		var counter atomic.Int32

		stop := Every(10*time.Millisecond, func() {
			counter.Add(1)
		})

		require.Eventually(t, func() bool {
			return counter.Load() >= 2
		}, 200*time.Millisecond, 5*time.Millisecond, "should have been called at least twice")

		// First stop
		stop()
		countAfterFirstStop := counter.Load()

		require.Never(t, func() bool {
			return counter.Load() > countAfterFirstStop
		}, 50*time.Millisecond, 5*time.Millisecond, "counter should not increase after first stop")

		// Second stop should be safe
		stop()
		countAfterSecondStop := counter.Load()

		assert.Equal(t, countAfterFirstStop, countAfterSecondStop,
			"second stop call should have no effect")
	})

	t.Run("stop returns immediately", func(t *testing.T) {

		stop := Every(100*time.Millisecond, func() {
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
			stop := Every(100*time.Millisecond, func() {})
			stop()
		}
	})

	b.Run("WithFunction", func(b *testing.B) {
		var counter atomic.Int64

		stop := Every(1*time.Microsecond, func() {
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
