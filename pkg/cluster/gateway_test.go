package cluster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNodeNameWithTimestamp(t *testing.T) {
	now := time.Now()
	name := nodeNameWithTimestamp("node-1", now)
	require.Contains(t, name, "node-1:")

	parsed := parseJoinTime(name)
	require.Equal(t, now.UnixNano(), parsed.UnixNano())
}

func TestParseJoinTime_Invalid(t *testing.T) {
	tests := []string{
		"no-colon",
		"trailing-colon:",
		"not-a-number:abc",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			jt := parseJoinTime(name)
			require.True(t, jt.IsZero())
		})
	}
}

func TestGatewayElection_OldestWins(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC) // 1 second later
	t3 := time.Date(2024, 1, 1, 0, 0, 2, 0, time.UTC) // 2 seconds later

	names := []string{
		nodeNameWithTimestamp("node-3", t3),
		nodeNameWithTimestamp("node-1", t1), // oldest
		nodeNameWithTimestamp("node-2", t2),
	}

	// Find oldest (same logic as evaluateGateway)
	var oldestName string
	var oldestTime time.Time

	for _, name := range names {
		jt := parseJoinTime(name)
		if oldestName == "" || jt.Before(oldestTime) {
			oldestName = name
			oldestTime = jt
		}
	}

	require.Contains(t, oldestName, "node-1")
}
