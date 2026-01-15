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

		// Initial time should be set synchronously
		now := clock.Now()

		// Should be within reasonable bounds of current time
		require.WithinDuration(t, time.Now(), now, 50*time.Millisecond)
	})

	t.Run("Now returns cached time within resolution", func(t *testing.T) {
		resolution := 50 * time.Millisecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		// Get initial time (set synchronously in constructor)
		t1 := clock.Now()

		// Verify time doesn't change within resolution
		require.Never(t, func() bool {
			return !clock.Now().Equal(t1)
		}, resolution/2, time.Millisecond, "cached time should not change within resolution")

		// Wait for resolution to pass, cached time should update
		require.Eventually(t, func() bool {
			return clock.Now().After(t1)
		}, 3*resolution, time.Millisecond, "cached time should update after resolution")
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

		// Wait for time to change at least once to ensure clock is running
		initialTime := clock.Now()
		require.Eventually(t, func() bool {
			return clock.Now().After(initialTime)
		}, 3*resolution, time.Millisecond, "clock should update")

		// Close the clock
		clock.Close()

		// Get the time after close
		t1 := clock.Now()

		// Verify time doesn't change after Close()
		require.Never(t, func() bool {
			return !clock.Now().Equal(t1)
		}, resolution*3, time.Millisecond, "time should not change after Close()")
	})

	t.Run("Very small resolution works", func(t *testing.T) {
		resolution := 100 * time.Microsecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		// Time is set synchronously, no sleep needed
		now := clock.Now()

		require.WithinDuration(t, time.Now(), now, 5*time.Millisecond)
	})

	t.Run("Large resolution works", func(t *testing.T) {
		resolution := 100 * time.Millisecond
		clock := NewCachedClock(resolution)
		defer clock.Close()

		// Get initial time (set synchronously in constructor)
		t1 := clock.Now()

		// Verify time doesn't change within resolution
		require.Never(t, func() bool {
			return !clock.Now().Equal(t1)
		}, resolution/2, time.Millisecond, "time should not change within large resolution")
	})

	t.Run("CachedClock implements Clock interface", func(t *testing.T) {
		var _ Clock = &CachedClock{} // nolint:exhaustruct
	})
}
