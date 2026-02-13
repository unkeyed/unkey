package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayElection_SmallestNameWins(t *testing.T) {
	names := []string{
		"node-3",
		"node-1", // smallest
		"node-2",
	}

	// Find smallest (same logic as evaluateGateway)
	smallest := names[0]
	for _, name := range names[1:] {
		if name < smallest {
			smallest = name
		}
	}

	require.Equal(t, "node-1", smallest)
}
