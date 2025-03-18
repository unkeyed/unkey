package ring

import (
	"context"
	"fmt"
	"testing"

	"github.com/gonum/stat"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// we don't need tags for this test.
type tags struct{}

const (
	Nodes = 10
	Runs  = 1_000_000
)

func TestRing(t *testing.T) {

	r, err := New[tags](Config{TokensPerNode: 256, Logger: logging.NewNoop()})
	require.NoError(t, err)

	for i := range Nodes {
		require.NoError(t, r.AddNode(context.Background(), Node[tags]{ID: fmt.Sprintf("node-%d", i), Tags: tags{}}))
	}

	// The `counters` map is used to keep track of the number of occurrences of each node in the
	// `nodes` slice returned by the `FindNodes` method. Each node's ID is used as the key in the
	// `counters` map, and the value associated with each key represents the count of how many times
	// that node appears in the `nodes` slice during the simulation of finding nodes for random keys.
	counters := make(map[string]int)
	for range Runs {
		key := ksuid.New().String()

		node, err := r.FindNode(key)
		require.NoError(t, err)
		c, ok := counters[node.ID]
		if !ok {
			counters[node.ID] = 1
		} else {
			counters[node.ID] = c + 1
		}
	}

	// Ensure each node is selected at least once
	require.Equal(t, len(counters), Nodes)

	// Now we calculate the min, max, mean, and standard deviation of the number of times each node
	// was selected in the `nodes` slice during the simulation of finding nodes for random keys.
	minimum := -1
	maximum := -1
	for k, v := range counters {

		t.Logf("%s: %d", k, v)
		if minimum == -1 || v < minimum {
			minimum = v
		}
		if maximum == -1 || v > maximum {
			maximum = v
		}
	}
	cs := make([]float64, len(counters))
	i := 0
	for _, v := range counters {
		cs[i] = float64(v)
		i++
	}
	m, s := stat.MeanStdDev(cs, nil)
	relStddev := s / m

	require.LessOrEqual(t, relStddev, 0.1, "relative std should be less than 0.1, got: %f", relStddev)
}

func TestAddingNodeAddsTokensToRing(t *testing.T) {

	tokensPerNode := 256
	r, err := New[tags](Config{TokensPerNode: tokensPerNode, Logger: logging.NewNoop()})

	require.NoError(t, err)
	require.Empty(t, r.tokens)

	for i := 1; i <= 10; i++ {
		err := r.AddNode(context.Background(), Node[tags]{ID: fmt.Sprintf("node-%d", i), Tags: tags{}})
		require.NoError(t, err)
		require.Len(t, r.tokens, tokensPerNode*i)
	}

}
func TestNodeCount(t *testing.T) {
	r, err := New[tags](Config{TokensPerNode: 256, Logger: logging.NewNoop()})
	require.NoError(t, err)

	// Initial ring should be empty
	require.Empty(t, r.nodes)
	require.Empty(t, r.Members())

	// Add 5 nodes and verify count after each
	for i := 1; i <= 5; i++ {
		err = r.AddNode(context.Background(), Node[tags]{ID: fmt.Sprintf("node-%d", i), Tags: tags{}})
		require.NoError(t, err)
		require.Len(t, r.nodes, i)
		require.Len(t, r.Members(), i)
	}

	// Add a duplicate node - should fail and count remains the same
	err = r.AddNode(context.Background(), Node[tags]{ID: "node-1", Tags: tags{}})
	require.Error(t, err)
	require.Len(t, r.nodes, 5)
	require.Len(t, r.Members(), 5)

	// Remove nodes and verify count
	for i := 5; i >= 1; i-- {
		err = r.RemoveNode(context.Background(), fmt.Sprintf("node-%d", i))
		require.NoError(t, err)
		require.Len(t, r.nodes, i-1)
		require.Len(t, r.Members(), i-1)
	}

	// Removing a non-existent node should not return an error
	err = r.RemoveNode(context.Background(), "non-existent-node")
	require.NoError(t, err)
	require.Empty(t, r.nodes)
}

func TestFindNodeEmpty(t *testing.T) {
	r, err := New[tags](Config{TokensPerNode: 256, Logger: logging.NewNoop()})
	require.NoError(t, err)

	// Finding a node in an empty ring should return an error
	_, err = r.FindNode("any-key")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ring is empty")
}

func TestNodeRemovalRebuild(t *testing.T) {
	r, err := New[tags](Config{TokensPerNode: 256, Logger: logging.NewNoop()})
	require.NoError(t, err)

	// Add 3 nodes
	for i := 1; i <= 3; i++ {
		addErr := r.AddNode(context.Background(), Node[tags]{ID: fmt.Sprintf("node-%d", i), Tags: tags{}})
		require.NoError(t, addErr)
	}

	// Get initial node count and token count
	initialNodeCount := len(r.nodes)
	initialTokenCount := len(r.tokens)
	require.Equal(t, 3, initialNodeCount)
	require.Equal(t, 3*256, initialTokenCount)

	// Remove a node
	err = r.RemoveNode(context.Background(), "node-2")
	require.NoError(t, err)

	// Verify node count decreased by 1
	require.Len(t, r.nodes, initialNodeCount-1)

	// Verify token count decreased by tokensPerNode
	require.Len(t, r.tokens, initialTokenCount-256)

	// Verify that all remaining tokens have valid node IDs
	nodeIDs := make(map[string]struct{})
	for _, node := range r.Members() {
		nodeIDs[node.ID] = struct{}{}
	}

	for _, token := range r.tokens {
		_, exists := nodeIDs[token.nodeID]
		require.True(t, exists, "Token references non-existent node ID: %s", token.nodeID)
	}
}
