package clock

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCachedClock(t *testing.T) {
	t.Run("NewCachedClock creates working clock", func(t *testing.T) {
		resolution := 10 * time.Millisecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		// Initial time should be set quickly
		time.Sleep(2 * time.Millisecond)
		now := clock.Now()

		// Should be within reasonable bounds of current time
		require.WithinDuration(t, time.Now(), now, 50*time.Millisecond)
	})

	t.Run("Now returns cached time within resolution", func(t *testing.T) {
		resolution := 50 * time.Millisecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		// Get initial time
		time.Sleep(resolution + 5*time.Millisecond) // Ensure cache is populated
		t1 := clock.Now()
		realTime1 := time.Now()

		// Wait less than resolution, time should be the same
		time.Sleep(10 * time.Millisecond)
		t2 := clock.Now()
		realTime2 := time.Now()

		// Cached time should be the same (within nanoseconds)
		require.True(t, t1.Equal(t2), "cached time should not change within resolution")

		// Real time should have advanced
		require.True(t, realTime2.After(realTime1), "real time should advance")

		// Wait for resolution to pass, cached time should update
		time.Sleep(resolution + 5*time.Millisecond)
		t3 := clock.Now()

		require.True(t, t3.After(t2), "cached time should update after resolution")
	})

	t.Run("Concurrent access is safe", func(t *testing.T) {
		resolution := 1 * time.Millisecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		const numGoroutines = 100
		const numCalls = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numCalls; j++ {
					_ = clock.Now()
				}
			}()
		}

		wg.Wait()
		// If we reach here without panic/race, concurrent access is safe
	})

	t.Run("Close stops background goroutine", func(t *testing.T) {
		resolution := 10 * time.Millisecond
		clock := NewCachedClock(resolution)

		// Let it run for a bit
		time.Sleep(resolution + 5*time.Millisecond)
		t1 := clock.Now()

		// Close the clock
		clock.Close()

		// Wait longer than resolution
		time.Sleep(resolution * 3)

		t2 := clock.Now()

		// Time should be the same after close (no more updates)
		require.True(t, t1.Equal(t2), "time should not change after Close()")
	})

	t.Run("Very small resolution works", func(t *testing.T) {
		resolution := 100 * time.Microsecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		time.Sleep(1 * time.Millisecond)
		now := clock.Now()

		require.WithinDuration(t, time.Now(), now, 5*time.Millisecond)
	})

	t.Run("Large resolution works", func(t *testing.T) {
		resolution := 100 * time.Millisecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		// Get initial time
		time.Sleep(resolution + 10*time.Millisecond)
		t1 := clock.Now()

		// Wait less than resolution
		time.Sleep(10 * time.Millisecond)
		t2 := clock.Now()

		// Should be the same cached value
		require.True(t, t1.Equal(t2), "time should not change within large resolution")
	})

	t.Run("CachedClock implements Clock interface", func(t *testing.T) {
		var _ Clock = &CachedClock{} // nolint:exhaustruct
	})
}
