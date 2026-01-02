package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRealClock(t *testing.T) {
	clock := New()
	before := time.Now()
	now := clock.Now()
	after := time.Now()

	require.False(t, now.Before(before), "time should not be before test start")
	require.False(t, now.After(after), "time should not be after test end")
}
