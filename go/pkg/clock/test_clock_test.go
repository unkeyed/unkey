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
