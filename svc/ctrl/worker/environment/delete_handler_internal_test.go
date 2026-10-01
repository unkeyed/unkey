package environment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTopologyRemovalPollInterval(t *testing.T) {
	want := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second,
		30 * time.Second,
	}

	for attempt, wantDelay := range want {
		require.Equal(t, wantDelay, topologyRemovalPollInterval(uint(attempt)))
	}
}
