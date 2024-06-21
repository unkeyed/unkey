package ring

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/gonum/stat"
	"github.com/stretchr/testify/require"
)

func TestRing(t *testing.T) {
	NODES := 128
	RUNS := 1000000

	r, err := New(Config{TokensPerNode: 256 * NODES})
	require.NoError(t, err)

	for i := 0; i < NODES; i++ {
		require.NoError(t, r.AddNode(Node{Id: fmt.Sprintf("node-%d", i), RpcAddr: ""}))
	}

	counters := make(map[string]int)
	for i := 0; i < RUNS; i++ {
		buf := make([]byte, 64)
		_, err := rand.Read(buf)
		require.NoError(t, err)
		key := string(buf)

		node, err := r.FindNode(key)
		require.NoError(t, err)
		c, ok := counters[node.Id]
		if !ok {
			counters[node.Id] = 1
		} else {
			counters[node.Id] = c + 1
		}

	}

	require.Equal(t, len(counters), NODES)

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

	require.LessOrEqual(t, relStddev, 0.05, "relative std should be less than 0.05, got: %f", relStddev)

}
