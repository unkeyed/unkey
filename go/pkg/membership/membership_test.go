package membership_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

var CLUSTER_SIZES = []int{3, 9}

func TestJoin2Nodes(t *testing.T) {

	freePort := port.New()

	m1, err := membership.New(membership.Config{
		NodeID:        "node_1",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		HttpPort:      freePort.Get(),
		RpcPort:       freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	m2, err := membership.New(membership.Config{
		NodeID:        "node_2",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		HttpPort:      freePort.Get(),
		RpcPort:       freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	err = m1.Start(&discovery.Static{Addrs: []string{}})
	require.NoError(t, err)
	m1Members, err := m1.Members()
	require.NoError(t, err)
	require.Len(t, m1Members, 1)

	err = m2.Start(&discovery.Static{Addrs: []string{fmt.Sprintf("%s:%d", m1.Self().Host, m1.Self().GossipPort)}})
	require.NoError(t, err)

	require.NoError(t, err)
	require.Eventually(t, func() bool {

		m1Members, err := m1.Members()
		require.NoError(t, err)
		t.Log(m1Members)
		return len(m1Members) == 2

	}, 10*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {

		m2Members, err := m2.Members()
		require.NoError(t, err)
		return len(m2Members) == 2

	}, 10*time.Second, 10*time.Millisecond)

}

func TestMembers_returns_all_members(t *testing.T) {
	for _, clusterSize := range CLUSTER_SIZES {

		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {

			nodes := runMany(t, clusterSize)

			for _, m := range nodes {
				members, err := m.Members()
				require.NoError(t, err)
				require.Len(t, members, clusterSize)

			}
		})
	}

}

// Test whether nodes correctly emit a join event when they join the cluster
// When a node joins, a listener to the `JoinEvents` topic should receive a message with the member that jioned.
func TestJoin_emits_join_event(t *testing.T) {
	for _, clusterSize := range CLUSTER_SIZES {
		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {
			freePort := port.New()
			members := make([]membership.Membership, clusterSize)

			// Store node IDs for later reference
			nodeIDs := make([]string, clusterSize)

			var err error
			for i := 0; i < clusterSize; i++ {
				nodeID := fmt.Sprintf("node_%d", i)
				nodeIDs[i] = nodeID

				members[i], err = membership.New(membership.Config{
					NodeID:        nodeID,
					AdvertiseHost: "127.0.0.1",
					GossipPort:    freePort.Get(),
					Logger:        logging.NewNoop(),
				})
				require.NoError(t, err)
			}

			joinEvents := members[0].SubscribeJoinEvents()
			joinMu := sync.RWMutex{}
			joinedNodes := make(map[string]bool) // Track by NodeID

			go func() {
				for event := range joinEvents {
					joinMu.Lock()
					joinedNodes[event.NodeID] = true // Store by NodeID
					joinMu.Unlock()
				}
			}()

			// First node is already "joined" to itself, so we need to track the others
			peerAddrs := make([]string, 0)
			for _, m := range members {
				err = m.Start(&discovery.Static{Addrs: peerAddrs})
				require.NoError(t, err)

				peerAddrs = append(peerAddrs, fmt.Sprintf("%s:%d", m.Self().Host, m.Self().GossipPort))
			}

			// Check if each node (except the first) has joined
			for i := 1; i < clusterSize; i++ {
				nodeID := nodeIDs[i]
				require.Eventually(t, func() bool {
					joinMu.RLock()
					hasJoined := joinedNodes[nodeID]
					joinMu.RUnlock()
					return hasJoined
				}, 30*time.Second, 10*time.Millisecond)
			}
		})
	}
}

// Test whether nodes correctly emit a leave event when they leave the cluster
// When a node leaves, a listener to the `LeaveEvents` topic should receive a message with the member that left.
func TestLeave_emits_leave_event(t *testing.T) {
	for _, clusterSize := range CLUSTER_SIZES {
		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {
			nodes := runMany(t, clusterSize)

			// Store nodeIDs for later reference
			nodeIDs := make([]string, clusterSize)
			for i, n := range nodes {
				nodeIDs[i] = n.Self().NodeID
			}

			leaveEvents := nodes[0].SubscribeLeaveEvents()
			leftMu := sync.RWMutex{}
			leftNodes := make(map[string]bool) // Track by NodeID

			go func() {
				for event := range leaveEvents {
					leftMu.Lock()
					leftNodes[event.NodeID] = true // Store by NodeID
					leftMu.Unlock()
				}
			}()

			for _, n := range nodes[1:] {
				err := n.Leave()
				require.NoError(t, err)
			}

			// Check each node (except the first) if it has left
			for i := 1; i < clusterSize; i++ {
				nodeID := nodeIDs[i]
				require.Eventually(t, func() bool {
					leftMu.RLock()
					hasLeft := leftNodes[nodeID]
					leftMu.RUnlock()
					return hasLeft
				}, 30*time.Second, 10*time.Millisecond)
			}
		})
	}
}
func TestRPCPortPropagation(t *testing.T) {
	freePort := port.New()

	// Create nodes with different RPC ports
	m1, err := membership.New(membership.Config{
		NodeID:        "rpc_node_1",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		RpcPort:       9001,
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	m2, err := membership.New(membership.Config{
		NodeID:        "rpc_node_2",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		RpcPort:       9002,
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		m1.Leave()
		m2.Leave()
	})

	// Start the nodes
	err = m1.Start(&discovery.Static{Addrs: []string{}})
	require.NoError(t, err)

	err = m2.Start(&discovery.Static{Addrs: []string{fmt.Sprintf("%s:%d", m1.Self().Host, m1.Self().GossipPort)}})
	require.NoError(t, err)

	// Wait for cluster to form
	require.Eventually(t, func() bool {
		members, membersErr := m1.Members()
		return membersErr == nil && len(members) == 2
	}, 5*time.Second, 10*time.Millisecond)

	// Check if m1 can see m2's RPC port
	members, err := m1.Members()
	require.NoError(t, err)

	// Find the m2 member
	var m2Member membership.Member
	for _, member := range members {
		if member.NodeID == "rpc_node_2" {
			m2Member = member
			break
		}
	}

	// Verify m2's RPC port is correctly propagated
	require.Equal(t, 9002, m2Member.RpcPort)
}

func TestRejoinAfterLeave(t *testing.T) {
	freePort := port.New()

	// Create a small cluster
	nodes := runMany(t, 3)
	t.Cleanup(func() {
		for _, node := range nodes {
			node.Leave()
		}
	})

	// Have one node leave
	leavingNode := nodes[1]
	nodeID := leavingNode.Self().NodeID
	err := leavingNode.Leave()
	require.NoError(t, err)

	// Wait for the node to be recognized as gone
	require.Eventually(t, func() bool {
		members, membersErr := nodes[0].Members()
		require.NoError(t, membersErr)
		return len(members) == 2
	}, 5*time.Second, 10*time.Millisecond)

	// Create a new node with the same ID
	rejoinNode, err := membership.New(membership.Config{
		NodeID:        nodeID,
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		rejoinNode.Leave()
	})

	// Try to rejoin
	err = rejoinNode.Start(&discovery.Static{
		Addrs: []string{fmt.Sprintf("%s:%d", nodes[0].Self().Host, nodes[0].Self().GossipPort)},
	})
	require.NoError(t, err)

	// Verify the node is back in the cluster
	require.Eventually(t, func() bool {
		members, err := nodes[0].Members()
		require.NoError(t, err)
		return len(members) == 3
	}, 5*time.Second, 10*time.Millisecond)
}
func TestConcurrentJoins(t *testing.T) {
	freePort := port.New()

	// Create a "seed" node
	seed, err := membership.New(membership.Config{
		NodeID:        "seed_node",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		RpcPort:       freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		seed.Leave()
	})

	err = seed.Start(&discovery.Static{Addrs: []string{}})
	require.NoError(t, err)

	// Number of nodes to join concurrently
	numNodes := 5
	nodes := make([]membership.Membership, numNodes)

	// WaitGroup to track completion
	var wg sync.WaitGroup
	wg.Add(numNodes)

	// Create and start nodes concurrently
	for i := 0; i < numNodes; i++ {
		go func() {
			defer wg.Done()

			node, err := membership.New(membership.Config{
				NodeID:        fmt.Sprintf("concurrent_node_%d", i),
				AdvertiseHost: "127.0.0.1",
				GossipPort:    freePort.Get(),
				RpcPort:       freePort.Get(),
				Logger:        logging.NewNoop(),
			})
			if err != nil {
				t.Errorf("Failed to create node %d: %v", i, err)
				return
			}

			nodes[i] = node

			// Join the seed node
			err = node.Start(&discovery.Static{
				Addrs: []string{fmt.Sprintf("%s:%d", seed.Self().Host, seed.Self().GossipPort)},
			})
			if err != nil {
				t.Errorf("Failed to start node %d: %v", i, err)
			}
		}()
	}

	// Wait for all nodes to join
	wg.Wait()

	// Cleanup all nodes
	t.Cleanup(func() {
		for _, node := range nodes {
			if node != nil {
				node.Leave()
			}
		}
	})

	// Wait for cluster to converge
	require.Eventually(t, func() bool {
		members, err := seed.Members()
		return err == nil && len(members) == numNodes+1 // +1 for seed node
	}, 10*time.Second, 10*time.Millisecond)

	// Verify all nodes can see each other
	for _, node := range nodes {
		if node == nil {
			continue
		}

		require.Eventually(t, func() bool {
			members, err := node.Members()
			if err != nil {
				return false
			}
			return len(members) == numNodes+1
		}, 5*time.Second, 10*time.Millisecond)
	}
}
func TestConsistentMemberList(t *testing.T) {
	// Create a cluster with 5 nodes
	nodes := runMany(t, 5)
	t.Cleanup(func() {
		for _, node := range nodes {
			node.Leave()
		}
	})

	// Get member lists from all nodes
	memberLists := make([][]membership.Member, len(nodes))
	for i, node := range nodes {
		members, err := node.Members()
		require.NoError(t, err)
		memberLists[i] = members
	}

	// Ensure all nodes see the same members (though order may differ)
	for i := 1; i < len(memberLists); i++ {
		require.Equal(t, len(memberLists[0]), len(memberLists[i]),
			"Node %d should see the same number of members as node 0", i)

		// Check if all members in list 0 exist in list i
		for _, m0 := range memberLists[0] {
			found := false
			for _, mi := range memberLists[i] {
				if m0.NodeID == mi.NodeID {
					found = true
					// Also check that member attributes match
					require.Equal(t, m0.RpcPort, mi.RpcPort,
						"Member %s should have same RpcPort across nodes", m0.NodeID)
					break
				}
			}
			require.True(t, found, "Member %s from node 0 not found in node %d", m0.NodeID, i)
		}
	}
}
func TestInvalidJoinAddresses(t *testing.T) {
	freePort := port.New()

	m, err := membership.New(membership.Config{
		NodeID:        "test_node",
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		m.Leave()
	})

	// Try to join non-existent nodes
	err = m.Start(&discovery.Static{
		Addrs: []string{
			"127.0.0.1:1",                 // Invalid port
			"nonexistent.host.local:8000", // Invalid hostname
		},
	})

	require.Error(t, err)

}
func TestSelfDiscovery(t *testing.T) {
	freePort := port.New()

	nodeID := "self_test_node"
	m, err := membership.New(membership.Config{
		NodeID:        nodeID,
		AdvertiseHost: "127.0.0.1",
		GossipPort:    freePort.Get(),
		RpcPort:       freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		m.Leave()
	})

	// Start the node
	err = m.Start(&discovery.Static{Addrs: []string{}})
	require.NoError(t, err)

	// Get self information two ways
	self := m.Self()

	// Get from members list
	members, err := m.Members()
	require.NoError(t, err)

	// Find self in members list
	var selfFromList membership.Member
	for _, member := range members {
		if member.NodeID == nodeID {
			selfFromList = member
			break
		}
	}

	// Verify both representations match
	require.Equal(t, self.NodeID, selfFromList.NodeID)
	require.Equal(t, self.Host, selfFromList.Host)
	require.Equal(t, self.GossipPort, selfFromList.GossipPort)
	require.Equal(t, self.RpcPort, selfFromList.RpcPort)
}

func TestMultipleNodesLeaving(t *testing.T) {
	// Create a cluster
	nodes := runMany(t, 5)
	t.Cleanup(func() {
		for _, node := range nodes {
			node.Leave()
		}
	})

	// Have multiple nodes leave simultaneously
	leaveWg := sync.WaitGroup{}
	leaveWg.Add(3)

	for i := 1; i < 4; i++ {
		go func() {
			defer leaveWg.Done()
			err := nodes[i].Leave()
			require.NoError(t, err)
		}()
	}

	leaveWg.Wait()

	// Verify the remaining nodes notice all departures
	require.Eventually(t, func() bool {
		members, err := nodes[0].Members()
		require.NoError(t, err)
		return len(members) == 2 // Only nodes[0] and nodes[4] should remain
	}, 5*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		members, err := nodes[4].Members()
		require.NoError(t, err)
		return len(members) == 2 // Only nodes[0] and nodes[4] should remain
	}, 5*time.Second, 10*time.Millisecond)
}
func runMany(t *testing.T, n int) []membership.Membership {

	freePort := port.New()

	members := make([]membership.Membership, n)
	for i := 0; i < n; i++ {

		// sometimes we had port collissions, that's why we're retrying this
		err := retry.New(retry.Attempts(10), retry.Backoff(func(n int) time.Duration {
			return 0
		})).Do(func() error {
			var mErr error
			members[i], mErr = membership.New(membership.Config{
				NodeID:        fmt.Sprintf("node_%d", i),
				AdvertiseHost: "127.0.0.1",
				GossipPort:    freePort.Get(),
				Logger:        logging.NewNoop(),
			})
			return mErr
		})
		require.NoError(t, err)

	}

	peerAddrs := make([]string, 0)
	for _, m := range members {
		err := m.Start(discovery.Static{Addrs: peerAddrs})
		require.NoError(t, err)
		peerAddrs = append(peerAddrs, fmt.Sprintf("%s:%d", m.Self().Host, m.Self().GossipPort))
	}

	for _, m := range members {
		require.Eventually(t, func() bool {
			members, err := m.Members()
			require.NoError(t, err)
			return len(members) == n
		}, 5*time.Second, 10*time.Millisecond)
	}
	return members

}
