package cluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
)

func TestClusterAddsSelfToRing(t *testing.T) {
	// Create cluster nodes with real membership
	freePort := port.New()

	// Get available ports for different services
	rpcPort := freePort.Get()
	gossipPort1 := freePort.Get()

	// Create the first membership system (will be used for our cluster node)
	membershipSys1, err := membership.New(membership.Config{
		NodeID:        "node-1",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    gossipPort1,
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	// Create a cluster using the membership system
	clusterNode := Node{
		ID:      "node-1",
		RpcAddr: fmt.Sprintf("127.0.0.1:%d", rpcPort),
	}

	cluster, err := New(Config{
		Self:       clusterNode,
		Membership: membershipSys1,
		Logger:     logging.NewNoop(),
	})
	require.NoError(t, err)

	// Start the membership system with empty discovery (standalone node)
	err = membershipSys1.Start(&discovery.Static{Addrs: []string{}})
	require.NoError(t, err)

	// Test if we can find the local node using its ID as a key
	// This is the key test - the self node must be in the ring
	node, err := cluster.FindNode(context.Background(), clusterNode.ID)
	require.NoError(t, err)
	require.Equal(t, clusterNode.ID, node.ID)

	// Get members and verify self is included
	members := cluster.ring.Members()
	require.Len(t, members, 1, "Ring should contain exactly one node (self)")
	require.Equal(t, clusterNode.ID, members[0].ID)

	// Create a second membership node
	gossipPort2 := freePort.Get()
	membershipSys2, err := membership.New(membership.Config{
		NodeID:        "node-2",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    gossipPort2,
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	// Start the second node with discovery pointing to first node
	err = membershipSys2.Start(&discovery.Static{Addrs: []string{fmt.Sprintf("127.0.0.1:%d", gossipPort1)}})
	require.NoError(t, err)

	// Wait for the memberlist to exchange information and for the ring to be updated
	require.Eventually(t, func() bool {
		return len(cluster.ring.Members()) == 2
	}, 5*time.Second, 100*time.Millisecond, "Ring should eventually contain both nodes")

	// Verify we can find both nodes by their IDs
	node1, err := cluster.FindNode(context.Background(), "node-1")
	require.NoError(t, err)
	require.Equal(t, "node-1", node1.ID)

	// When we look for node-2, we should get its info
	// This depends on the consistent hash working correctly
	// In a small cluster with limited keys, test behavior depends on hash distribution
	// This is somewhat of a flaky test (keys might hash to other nodes)

	// Test that the cluster members match the expected nodes
	members = cluster.ring.Members()
	require.Len(t, members, 2)

	// Check specific member properties
	nodeIds := make(map[string]bool)
	for _, member := range members {
		nodeIds[member.ID] = true
	}
	require.True(t, nodeIds["node-1"], "Ring should contain node-1")
	require.True(t, nodeIds["node-2"], "Ring should contain node-2")

	// Test shutdown
	err = cluster.Shutdown(context.Background())
	require.NoError(t, err)

	// Also leave the second membership, just to clean up
	err = membershipSys2.Leave()
	require.NoError(t, err)
}

func TestNoopCluster(t *testing.T) {
	// Create a noop cluster
	noop := NewNoop("test-node", "localhost")

	// Test FindNode always returns self
	node, err := noop.FindNode(context.Background(), "any-key")
	require.NoError(t, err)
	require.Equal(t, "test-node", node.ID)
	require.Equal(t, "", node.RpcAddr)

	// Test Self returns the right node
	self := noop.Self()
	require.Equal(t, "test-node", self.ID)

	joins := noop.SubscribeJoin()
	leaves := noop.SubscribeLeave()

	err = noop.Shutdown(context.Background())
	require.NoError(t, err)

	// Test subscription channels don't close or send
	select {
	case <-joins:
		t.Fatal("Expected join channel to not send any events")
	case <-leaves:
		t.Fatal("Expected leave channel to not send any events")
	case <-time.After(time.Second):
		// This is the expected behavior
	}
}
