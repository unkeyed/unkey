package cluster

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
)

func testMessage(key string) *clusterv1.ClusterMessage_CacheInvalidation {
	return &clusterv1.ClusterMessage_CacheInvalidation{
		CacheInvalidation: &cachev1.CacheInvalidationEvent{
			CacheName: "test",
			Action:    &cachev1.CacheInvalidationEvent_CacheKey{CacheKey: key},
		},
	}
}

func TestCluster_SingleNode_BroadcastAndReceive(t *testing.T) {
	t.Run("broadcast receives on single node", func(t *testing.T) {
		var received atomic.Int32

		c, err := New(Config{
			Region:   "us-east-1",
			NodeID:   "test-node-1",
			BindAddr: "127.0.0.1",
			OnMessage: func(msg *clusterv1.ClusterMessage) {
				received.Add(1)
			},
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, c.Close()) }()

		// Single node should be bridge
		require.Eventually(t, func() bool {
			return c.IsBridge()
		}, 2*time.Second, 50*time.Millisecond, "single node should become bridge")

		require.Len(t, c.Members(), 1, "should have 1 member")

		require.NoError(t, c.Broadcast(testMessage("hello")))

		require.Eventually(t, func() bool {
			return received.Load() >= 1
		}, 5*time.Second, 50*time.Millisecond, "handler should have received the broadcast")
	})
}

func TestCluster_MultiNode_BroadcastDelivery(t *testing.T) {
	const nodeCount = 3
	var clusters []Cluster
	var received [nodeCount]atomic.Int32

	// Create first node
	c1, err := New(Config{
		Region:   "us-east-1",
		NodeID:   "node-0",
		BindAddr: "127.0.0.1",
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			received[0].Add(1)
		},
	})
	require.NoError(t, err)
	clusters = append(clusters, c1)

	// Get the first node's address for seeding
	c1Addr := c1.Members()[0].FullAddress().Addr

	// Create remaining nodes, seeding with first node
	for i := 1; i < nodeCount; i++ {
		idx := i
		// Delay to ensure deterministic ordering for bridge election
		time.Sleep(50 * time.Millisecond)

		cn, createErr := New(Config{
			Region:   "us-east-1",
			NodeID:   fmt.Sprintf("node-%d", idx),
			BindAddr: "127.0.0.1",
			LANSeeds: []string{c1Addr},
			OnMessage: func(msg *clusterv1.ClusterMessage) {
				received[idx].Add(1)
			},
		})
		require.NoError(t, createErr)
		clusters = append(clusters, cn)
	}

	defer func() {
		for i := len(clusters) - 1; i >= 0; i-- {
			require.NoError(t, clusters[i].Close())
		}
	}()

	// Wait for all nodes to see each other
	require.Eventually(t, func() bool {
		for _, c := range clusters {
			if len(c.Members()) != nodeCount {
				return false
			}
		}
		return true
	}, 5*time.Second, 100*time.Millisecond, "all nodes should see each other")

	// Wait for bridge election to settle
	require.Eventually(t, func() bool {
		bridgeCount := 0
		for _, c := range clusters {
			if c.IsBridge() {
				bridgeCount++
			}
		}
		return bridgeCount == 1
	}, 5*time.Second, 100*time.Millisecond, "exactly one node should be bridge")

	// The first node (oldest) should be bridge
	require.True(t, clusters[0].IsBridge(), "oldest node should be bridge")

	t.Run("broadcast delivers to all", func(t *testing.T) {
		require.NoError(t, clusters[0].Broadcast(testMessage("multi-node-hello")))

		for i := 0; i < nodeCount; i++ {
			idx := i
			require.Eventually(t, func() bool {
				return received[idx].Load() >= 1
			}, 5*time.Second, 50*time.Millisecond, "node %d should have received the broadcast", idx)
		}
	})
}

func TestCluster_BridgeFailover(t *testing.T) {
	// Create first node (will be bridge)
	var recv1, recv2 atomic.Int32

	c1, err := New(Config{
		Region:   "us-east-1",
		NodeID:   "node-1",
		BindAddr: "127.0.0.1",
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			recv1.Add(1)
		},
	})
	require.NoError(t, err)

	c1Addr := c1.Members()[0].FullAddress().Addr

	// Delay to ensure c1 is older
	time.Sleep(50 * time.Millisecond)

	c2, err := New(Config{
		Region:   "us-east-1",
		NodeID:   "node-2",
		BindAddr: "127.0.0.1",
		LANSeeds: []string{c1Addr},
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			recv2.Add(1)
		},
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, c2.Close()) }()

	// Wait for both to see each other
	require.Eventually(t, func() bool {
		return len(c1.Members()) == 2 && len(c2.Members()) == 2
	}, 5*time.Second, 100*time.Millisecond)

	// Wait for bridge to settle: c1 should be bridge (oldest)
	require.Eventually(t, func() bool {
		return c1.IsBridge() && !c2.IsBridge()
	}, 5*time.Second, 100*time.Millisecond, "c1 should be bridge, c2 should not")

	// Kill c1 (the bridge)
	require.NoError(t, c1.Close())

	// c2 should become bridge
	require.Eventually(t, func() bool {
		return c2.IsBridge()
	}, 10*time.Second, 100*time.Millisecond, "c2 should become bridge after c1 leaves")
}

func TestCluster_MultiRegion_WANBroadcast(t *testing.T) {
	var recvA, recvB atomic.Int32
	var muB sync.Mutex
	var lastKeyB string

	// --- Region A: single node (auto-promotes to bridge) ---
	nodeA, err := New(Config{
		Region:   "us-east-1",
		NodeID:   "node-a",
		BindAddr: "127.0.0.1",
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			recvA.Add(1)
		},
	})
	require.NoError(t, err)

	// Wait for node A to become bridge
	require.Eventually(t, func() bool {
		return nodeA.IsBridge()
	}, 5*time.Second, 50*time.Millisecond, "node A should become bridge")

	// Get node A's WAN address (assigned after promotion)
	var wanAddrA string
	require.Eventually(t, func() bool {
		wanAddrA = nodeA.WANAddr()
		return wanAddrA != ""
	}, 5*time.Second, 50*time.Millisecond, "node A WAN address should be available")

	// --- Region B: single node, seeds WAN with region A's bridge ---
	nodeB, err := New(Config{
		Region:   "eu-west-1",
		NodeID:   "node-b",
		BindAddr: "127.0.0.1",
		WANSeeds: []string{wanAddrA},
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			muB.Lock()
			lastKeyB = msg.GetCacheInvalidation().GetCacheKey()
			muB.Unlock()
			recvB.Add(1)
		},
	})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, nodeB.Close())
		require.NoError(t, nodeA.Close())
	}()

	// Wait for node B to become bridge
	require.Eventually(t, func() bool {
		return nodeB.IsBridge()
	}, 5*time.Second, 50*time.Millisecond, "node B should become bridge")

	// Wait for WAN pools to see each other (each bridge sees 2 WAN members)
	implA := nodeA.(*gossipCluster)
	implB := nodeB.(*gossipCluster)
	require.Eventually(t, func() bool {
		implA.mu.RLock()
		wanA := implA.wan
		implA.mu.RUnlock()

		implB.mu.RLock()
		wanB := implB.wan
		implB.mu.RUnlock()

		if wanA == nil || wanB == nil {
			return false
		}
		return wanA.NumMembers() == 2 && wanB.NumMembers() == 2
	}, 10*time.Second, 100*time.Millisecond, "WAN pools should see each other")

	// Broadcast from region A
	require.NoError(t, nodeA.Broadcast(testMessage("cross-region-hello")))

	// Verify region B receives it via the WAN relay
	require.Eventually(t, func() bool {
		return recvB.Load() >= 1
	}, 10*time.Second, 100*time.Millisecond, "node B should receive cross-region broadcast")

	muB.Lock()
	require.Equal(t, "cross-region-hello", lastKeyB)
	muB.Unlock()
}

func TestCluster_MultiRegion_BidirectionalBroadcast(t *testing.T) {
	var muA, muB sync.Mutex
	var msgsA, msgsB []string

	// --- Region A ---
	nodeA, err := New(Config{
		Region:   "us-east-1",
		NodeID:   "node-a",
		BindAddr: "127.0.0.1",
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			muA.Lock()
			msgsA = append(msgsA, msg.GetCacheInvalidation().GetCacheKey())
			muA.Unlock()
		},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return nodeA.IsBridge() && nodeA.WANAddr() != ""
	}, 5*time.Second, 50*time.Millisecond)

	wanAddrA := nodeA.WANAddr()

	// --- Region B ---
	nodeB, err := New(Config{
		Region:   "eu-west-1",
		NodeID:   "node-b",
		BindAddr: "127.0.0.1",
		WANSeeds: []string{wanAddrA},
		OnMessage: func(msg *clusterv1.ClusterMessage) {
			muB.Lock()
			msgsB = append(msgsB, msg.GetCacheInvalidation().GetCacheKey())
			muB.Unlock()
		},
	})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, nodeB.Close())
		require.NoError(t, nodeA.Close())
	}()

	require.Eventually(t, func() bool {
		return nodeB.IsBridge()
	}, 5*time.Second, 50*time.Millisecond)

	// Wait for WAN connectivity
	implA := nodeA.(*gossipCluster)
	implB := nodeB.(*gossipCluster)
	require.Eventually(t, func() bool {
		implA.mu.RLock()
		wanA := implA.wan
		implA.mu.RUnlock()

		implB.mu.RLock()
		wanB := implB.wan
		implB.mu.RUnlock()

		if wanA == nil || wanB == nil {
			return false
		}
		return wanA.NumMembers() == 2 && wanB.NumMembers() == 2
	}, 10*time.Second, 100*time.Millisecond, "WAN pools should connect")

	// Broadcast from A → B
	require.NoError(t, nodeA.Broadcast(testMessage("from-east")))

	require.Eventually(t, func() bool {
		muB.Lock()
		defer muB.Unlock()
		for _, m := range msgsB {
			if m == "from-east" {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "B should receive message from A")

	// Broadcast from B → A
	require.NoError(t, nodeB.Broadcast(testMessage("from-west")))

	require.Eventually(t, func() bool {
		muA.Lock()
		defer muA.Unlock()
		for _, m := range msgsA {
			if m == "from-west" {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "A should receive message from B")
}

func TestCluster_Noop(t *testing.T) {
	c := NewNoop()

	require.False(t, c.IsBridge())
	require.Nil(t, c.Members())
	require.NoError(t, c.Broadcast(testMessage("test")))
	require.NoError(t, c.Close())
}
