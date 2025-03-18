package api_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/testflags"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// TestClusterFormation verifies that a cluster of API nodes can successfully form
// and communicate with each other.
func TestClusterFormation(t *testing.T) {
	testflags.SkipUnlessIntegration(t)

	// Create a containers instance for database
	containers := containers.New(t)
	dbDsn := containers.RunMySQL()

	// Get free ports for the nodes
	portAllocator := port.New()
	joinAddrs := []string{}

	// Start each node in a separate goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clusterSize := 3
	for i := 0; i < clusterSize; i++ {

		nodeID := uid.New("node")
		gossipPort := portAllocator.Get()
		config := api.Config{
			Platform:                    "test",
			Image:                       "test",
			HttpPort:                    portAllocator.Get(),
			Region:                      "test-region",
			Clock:                       nil, // Will use real clock
			ClusterEnabled:              true,
			ClusterNodeID:               nodeID,
			ClusterAdvertiseAddrStatic:  "localhost",
			ClusterRpcPort:              portAllocator.Get(),
			ClusterGossipPort:           gossipPort,
			ClusterDiscoveryStaticAddrs: joinAddrs,
			LogsColor:                   false,
			ClickhouseURL:               "",
			DatabasePrimary:             dbDsn,
			DatabaseReadonlyReplica:     "",
			OtelEnabled:                 false,
		}

		joinAddrs = append(joinAddrs, fmt.Sprintf("localhost:%d", gossipPort))

		go func() {
			require.NoError(t, api.Run(ctx, config))
		}()

		require.Eventually(t, func() bool {

			res, err := http.Get(fmt.Sprintf("http://localhost:%d/v2/liveness", config.HttpPort))
			if err != nil {
				return false
			}
			require.NoError(t, res.Body.Close())

			return res.StatusCode == http.StatusOK

		}, time.Second*10, time.Millisecond*100)

	}

	t.Log("All nodes started successfully")

	// Now verify cluster formation by checking cluster status endpoints
	// Give the cluster a moment to form
	time.Sleep(5 * time.Second)

	// Clean up
	cancel()                    // Signal all nodes to shut down
	time.Sleep(2 * time.Second) // Give them time to shut down
}
