package cluster

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

var CLUSTER_SIZES = []int{3, 9}

func TestMembershipChangesArePropagatedToHashRing(t *testing.T) {

	for _, clusterSize := range CLUSTER_SIZES {

		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {

			clusters := []*cluster{}

			// Starting clusters
			for i := 1; i <= clusterSize; i++ {
				c := createCluster(t, fmt.Sprintf("node_%d", i))
				clusters = append(clusters, c)
				addrs := []string{}
				for _, c := range clusters {
					addrs = append(addrs, c.membership.SerfAddr())
				}
				_, err := c.membership.Join(addrs...)
				require.NoError(t, err)

				// Check if the hash rings are updated
				for _, peer := range clusters {

					require.Eventually(t, func() bool {

						t.Logf("%s, clusters: %d, peer.ring.Members(): %d", peer.id, len(clusters), len(peer.ring.Members()))

						return len(peer.ring.Members()) == len(clusters)
					}, time.Minute, 100*time.Millisecond)
				}
			}

			// Stopping clusters

			for len(clusters) > 0 {

				i := rand.Intn(len(clusters))

				c := clusters[i]

				err := c.membership.Leave()
				require.NoError(t, err)
				clusters = append(clusters[:i], clusters[i+1:]...)

				// Check if the hash rings are updated
				for _, peer := range clusters {

					require.Eventually(t, func() bool {
						t.Logf("%s, clusters: %d, peer.ring.Members(): %d", peer.id, len(clusters), len(peer.ring.Members()))

						return len(peer.ring.Members()) == len(clusters)
					}, 5*time.Minute, 100*time.Millisecond)
				}

			}

		})
	}

}

func createCluster(t *testing.T, nodeId string) *cluster {
	t.Helper()

	logger := logging.New(nil).With().Str("nodeId", nodeId).Logger().Level(zerolog.ErrorLevel)
	rpcAddr := fmt.Sprintf("http://localhost:%d", port.Get())

	m, err := membership.New(membership.Config{
		NodeId: nodeId,
		Logger: logger,

		SerfAddr: fmt.Sprintf("localhost:%d", port.Get()),
		RpcAddr:  rpcAddr,
	})
	require.NoError(t, err)

	c, err := New(Config{
		NodeId:     nodeId,
		Membership: m,
		Logger:     logger,
		Metrics:    metrics.NewNoop(),
		Debug:      true,
		RpcAddr:    rpcAddr,
		AuthToken:  "test-auth-token",
	})
	require.NoError(t, err)

	return c

}

func TestFindNodeIsConsistent(t *testing.T) {

	for _, clusterSize := range CLUSTER_SIZES {

		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {

			clusters := []*cluster{}

			// Starting clusters
			for i := 1; i <= clusterSize; i++ {
				c := createCluster(t, fmt.Sprintf("node_%d", i))
				clusters = append(clusters, c)
				addrs := []string{}
				for _, c := range clusters {
					addrs = append(addrs, c.membership.SerfAddr())
				}
				_, err := c.membership.Join(addrs...)
				require.NoError(t, err)
			}

			// key -> nodeId -> count
			counters := make(map[string]map[string]int)

			keys := make([]string, 10000)
			for i := range keys {
				keys[i] = fmt.Sprintf("key-%d", i)
			}

			// Run the simulation
			for i := 0; i < 1_000_000; i++ {
				key := util.RandomElement(keys)
				node := util.RandomElement(clusters)
				found, err := node.FindNode(key)
				require.NoError(t, err)
				counter, ok := counters[key]
				if !ok {
					counter = make(map[string]int)
					counters[key] = counter
				}
				_, ok = counter[found.Id]
				if !ok {
					counter[found.Id] = 0
				}
				counter[found.Id]++

			}
			// t.Logf("counters: %+v", counters)

			for _, foundNodes := range counters {
				require.Len(t, foundNodes, 1)
			}

		})
	}

}
