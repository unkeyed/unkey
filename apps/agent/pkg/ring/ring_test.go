package ring

import (
	"fmt"
	"testing"

	"github.com/gonum/stat"
	"github.com/rs/zerolog"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

// we don't need tags for this test.
type tags struct{}

func TestRing(t *testing.T) {
	NODES := 128
	RUNS := 1000000

	r, err := New[tags](Config{TokensPerNode: 256, Logger: logging.New(nil).Level(zerolog.ErrorLevel)})
	require.NoError(t, err)

	for i := range NODES {
		require.NoError(t, r.AddNode(Node[tags]{Id: fmt.Sprintf("node-%d", i), Tags: tags{}}))
	}

	// The `counters` map is used to keep track of the number of occurrences of each node in the
	// `nodes` slice returned by the `FindNodes` method. Each node's ID is used as the key in the
	// `counters` map, and the value associated with each key represents the count of how many times
	//that node appears in the `nodes` slice during the simulation of finding nodes for random keys.
	counters := make(map[string]int)
	for range RUNS {
		key := ksuid.New().String()

		node, err := r.FindNode(key)
		require.NoError(t, err)
		c, ok := counters[node.Id]
		if !ok {
			counters[node.Id] = 1
		} else {
			counters[node.Id] = c + 1
		}
	}

	// Ensure each node is selected at least once
	require.Equal(t, len(counters), NODES)

	// Now we calculate the min, max, mean, and standard deviation of the number of times each node
	// was selected in the `nodes` slice during the simulation of finding nodes for random keys.
	min := -1
	max := -1
	for _, v := range counters {
		if min == -1 || v < min {
			min = v
		}
		if max == -1 || v > max {
			max = v
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

	fmt.Printf("min: %d, max: %d, mean: %f, stddev: %f, relstddev: %f\n", min, max, m, s, relStddev)
	require.LessOrEqual(t, relStddev, 0.1, "relative std should be less than 0.1, got: %f", relStddev)
}

func TestAddingNodeAddsTokensToRing(t *testing.T) {

	tokensPerNode := 256
	r, err := New[tags](Config{TokensPerNode: tokensPerNode, Logger: logging.New(nil).Level(zerolog.ErrorLevel)})

	require.NoError(t, err)
	require.Empty(t, r.tokens)

	for i := 1; i <= 10; i++ {
		err := r.AddNode(Node[tags]{Id: fmt.Sprintf("node-%d", i), Tags: tags{}})
		require.NoError(t, err)
		require.Len(t, r.tokens, tokensPerNode*i)
	}

}
