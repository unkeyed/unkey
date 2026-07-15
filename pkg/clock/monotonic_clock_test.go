package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMonotonicClock_Monotonic(t *testing.T) {
	c := NewMonotonic()
	first := c.Now()

	// Busy-wait a tiny amount so monotonic advances.
	for i := 0; i < 1000; i++ {
		_ = c.Now()
	}

	second := c.Now()
	require.GreaterOrEqual(t, second.UnixNano(), first.UnixNano(),
		"monotonic clock must never regress")
}

func TestMonotonicClock_AnchoredAtConstruction(t *testing.T) {
	before := time.Now()
	c := NewMonotonic()
	after := time.Now()

	first := c.Now()
	// First reading should be very close to construction time, bounded by
	// the timing of the before/after samples.
	require.GreaterOrEqual(t, first.UnixNano(), before.UnixNano())
	require.LessOrEqual(t, first.UnixNano(), after.Add(10*time.Millisecond).UnixNano())
}
