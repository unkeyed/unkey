package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTestClock(t *testing.T) {
	t.Run("NewTestClock with no args uses current time", func(t *testing.T) {
		before := time.Now()
		clock := NewTestClock()
		after := time.Now()

		now := clock.Now()
		require.False(t, now.Before(before), "time should not be before test start")
		require.False(t, now.After(after), "time should not be after test end")
	})

	t.Run("NewTestClock with specific time", func(t *testing.T) {
		specificTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		clock := NewTestClock(specificTime)

		require.True(t, clock.Now().Equal(specificTime))
	})

	t.Run("Tick advances time correctly", func(t *testing.T) {
		startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		clock := NewTestClock(startTime)

		duration := 5 * time.Minute
		newTime := clock.Tick(duration)
		expected := startTime.Add(duration)

		require.True(t, newTime.Equal(expected))
		require.True(t, clock.Now().Equal(expected))
	})

	t.Run("Set changes time correctly", func(t *testing.T) {
		clock := NewTestClock()
		newTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

		returnedTime := clock.Set(newTime)

		require.True(t, returnedTime.Equal(newTime))
		require.True(t, clock.Now().Equal(newTime))
	})

	t.Run("Multiple operations in sequence", func(t *testing.T) {
		startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		clock := NewTestClock(startTime)

		// Tick forward
		clock.Tick(time.Hour)
		expected := startTime.Add(time.Hour)
		require.True(t, clock.Now().Equal(expected))

		// Set to new time
		newTime := time.Date(2023, 2, 1, 12, 0, 0, 0, time.UTC)
		clock.Set(newTime)
		require.True(t, clock.Now().Equal(newTime))

		// Tick again
		clock.Tick(30 * time.Minute)
		expected = newTime.Add(30 * time.Minute)
		require.True(t, clock.Now().Equal(expected))
	})
}

func TestTestClockWithDifferentTimeZones(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, nyc)
	clock := NewTestClock(startTime)

	require.True(t, clock.Now().Equal(startTime))
	require.Equal(t, nyc, clock.Now().Location())
}

func TestTestClockWithNegativeTick(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewTestClock(startTime)

	newTime := clock.Tick(-1 * time.Hour)
	expected := startTime.Add(-1 * time.Hour)

	require.True(t, newTime.Equal(expected))
}

func TestClockInterface(t *testing.T) {
	var realClock Clock = &RealClock{}
	var testClock Clock = &TestClock{} // nolint:exhaustruct

	require.Implements(t, (*Clock)(nil), realClock)
	require.Implements(t, (*Clock)(nil), testClock)
}

func TestTestClockTicker(t *testing.T) {
	t.Run("Tick fires due ticks in order", func(t *testing.T) {
		clk := NewTestClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		ticker := clk.NewTicker(10 * time.Second)
		defer ticker.Stop()

		got := make(chan time.Time, 6)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range 6 {
				got <- <-ticker.C()
			}
		}()

		clk.Tick(60 * time.Second)
		<-done

		require.Len(t, got, 6)
		for i := 1; i <= 6; i++ {
			expected := time.Date(2024, 1, 1, 0, 0, 10*i, 0, time.UTC)
			require.True(t, expected.Equal(<-got),
				"tick %d should fire at offset %ds", i, 10*i)
		}
	})

	t.Run("Set jumping forward fires due ticks", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		clk := NewTestClock(start)
		ticker := clk.NewTicker(time.Second)
		defer ticker.Stop()

		got := 0
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range 3 {
				<-ticker.C()
				got++
			}
		}()

		clk.Set(start.Add(3 * time.Second))
		<-done
		require.Equal(t, 3, got)
	})

	t.Run("Set jumping backward does not fire", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		clk := NewTestClock(start)
		ticker := clk.NewTicker(time.Second)
		defer ticker.Stop()

		// Forward fires are synchronous, so a backward Set that fired
		// would have queued a value on the unbuffered channel by the
		// time Set returned. A non-blocking receive after Set proves
		// nothing was sent.
		clk.Set(start.Add(-time.Hour))
		select {
		case <-ticker.C():
			t.Fatal("ticker should not fire on backward Set")
		default:
		}
	})

	t.Run("Stop prevents further ticks", func(t *testing.T) {
		clk := NewTestClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		ticker := clk.NewTicker(time.Second)

		first := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			<-ticker.C()
			close(first)
			// A second tick after Stop would arrive on ticker.C();
			// the timeout case is the success path.
			select {
			case <-ticker.C():
				t.Errorf("ticker fired after Stop")
			case <-time.After(50 * time.Millisecond):
			}
		}()

		clk.Tick(time.Second)
		<-first
		ticker.Stop()
		clk.Tick(5 * time.Second)
		<-done
	})

	t.Run("NewTicker panics on non-positive duration", func(t *testing.T) {
		clk := NewTestClock()
		require.Panics(t, func() { clk.NewTicker(0) })
		require.Panics(t, func() { clk.NewTicker(-time.Second) })
	})
}
