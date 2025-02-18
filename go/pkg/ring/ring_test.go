package ring

import (
	"context"
	"fmt"
	"testing"

	"github.com/gonum/stat"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/logging"
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

	fmt.Printf("min: %d, max: %d, mean: %f, stddev: %f, relstddev: %f\n", minimum, maximum, m, s, relStddev)
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
