package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// ClusterNode represents a running instance of the API server
type ClusterNode struct {
	InstanceID string
	HttpPort   int
	RPCPort    int
	GossipPort int
}

// Harness is a test harness for creating and managing a cluster of API nodes
type Harness struct {
	t            *testing.T
	ctx          context.Context
	cancel       context.CancelFunc
	nodes        []ClusterNode
	ports        *port.FreePort
	containerMgr *containers.Containers
	Seed         *seed.Seeder
	dbDSN        string
	DB           db.Database
}

// Config contains configuration options for the test harness
type Config struct {
	// NumNodes is the number of API nodes to create in the cluster
	NumNodes int
}

// New creates a new cluster test harness
func New(t *testing.T, config Config) *Harness {
	t.Helper()

	require.Greater(t, config.NumNodes, 0)
	ctx, cancel := context.WithCancel(context.Background())

	containerMgr := containers.New(t)

	dbDSN := containerMgr.RunMySQL()
	db, err := db.New(db.Config{
		Logger:      logging.NewNoop(),
		PrimaryDSN:  dbDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	h := &Harness{
		t:            t,
		ctx:          ctx,
		cancel:       cancel,
		ports:        port.New(),
		containerMgr: containerMgr,
		nodes:        []ClusterNode{},
		Seed:         seed.New(t, db),
		dbDSN:        dbDSN,
		DB:           db,
	}

	t.Cleanup(func() {
		h.t.Log("Shutting down test cluster...")
		h.cancel()
	})

	h.Seed.Seed(ctx)

	// Prepare data for gossip-based cluster discovery
	var joinAddrs []string

	// Create and start each node
	for i := 0; i < config.NumNodes; i++ {
		node := h.createNode(i, joinAddrs)

		// Add this node's gossip address to the joinAddrs for subsequent nodes
		joinAddrs = append(joinAddrs, fmt.Sprintf("localhost:%d", node.GossipPort))
		h.nodes = append(h.nodes, node)
	}
	return h
}

func (h *Harness) Resources() seed.Resources {
	return h.Seed.Resources
}

// createNode creates and starts a single API node
func (h *Harness) createNode(index int, joinAddrs []string) ClusterNode {
	h.t.Helper()

	instanceID := fmt.Sprintf("i_%d", index)
	httpPort := h.ports.Get()
	rpcPort := h.ports.Get()
	gossipPort := h.ports.Get()

	nodeConfig := api.Config{
		Platform:                           "test",
		Image:                              "test",
		HttpPort:                           httpPort,
		Region:                             "test-region",
		Clock:                              nil, // Will use real clock
		ClusterEnabled:                     true,
		ClusterInstanceID:                  instanceID,
		ClusterAdvertiseAddrStatic:         "localhost",
		ClusterRpcPort:                     rpcPort,
		ClusterGossipPort:                  gossipPort,
		ClusterDiscoveryStaticAddrs:        joinAddrs,
		ClusterDiscoveryRedisURL:           "",
		ClusterAdvertiseAddrAwsEcsMetadata: false,
		DatabasePrimary:                    h.dbDSN,
		DatabaseReadonlyReplica:            "",
		LogsColor:                          false,
		ClickhouseURL:                      "",
		OtelEnabled:                        os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "",
	}

	// Start the node in a separate goroutine
	go func() {
		if err := api.Run(h.ctx, nodeConfig); err != nil {
			// If this is a planned shutdown (context canceled), don't fail the test
			if h.ctx.Err() == nil {
				h.t.Errorf("Node %s failed to run: %v", instanceID, err)
			}
		}
	}()

	// Ensure the node is up and running
	require.Eventually(h.t, func() bool {
		res, err := http.Get(fmt.Sprintf("http://localhost:%d/v2/liveness", httpPort))
		if err != nil {
			return false
		}
		defer res.Body.Close()
		return res.StatusCode == http.StatusOK
	}, 15*time.Second, 100*time.Millisecond, "API node %s failed to start", instanceID)

	h.t.Logf("Node %s started and healthy", instanceID)

	return ClusterNode{
		InstanceID: instanceID,
		HttpPort:   httpPort,
		RPCPort:    rpcPort,
		GossipPort: gossipPort,
	}
}

// GetNodes returns all nodes in the cluster
func (h *Harness) GetNodes() []ClusterNode {
	return h.nodes
}

// GetNode returns a specific node by index
func (h *Harness) GetNode(index int) ClusterNode {
	if index < 0 || index >= len(h.nodes) {
		h.t.Fatalf("Invalid node index: %d", index)
	}
	return h.nodes[index]
}
