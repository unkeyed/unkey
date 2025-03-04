package membership_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

var CLUSTER_SIZES = []int{3, 9, 36}

func TestJoin2Nodes(t *testing.T) {

	freePort := port.New()

	m1, err := membership.New(membership.Config{
		NodeID:        "node_1",
		AdvertiseAddr: "127.0.0.1",
		GossipPort:    freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	m2, err := membership.New(membership.Config{
		NodeID:        "node_2",
		AdvertiseAddr: "127.0.0.1",
		GossipPort:    freePort.Get(),
		Logger:        logging.NewNoop(),
	})
	require.NoError(t, err)

	err = m1.Start(&discovery.Static{Addrs: []string{}})
	require.NoError(t, err)
	m1Members, err := m1.Members()
	require.NoError(t, err)
	require.Len(t, m1Members, 1)

	err = m2.Start(&discovery.Static{Addrs: []string{m1.Self().Addr}})
	require.NoError(t, err)

	require.NoError(t, err)
	require.Eventually(t, func() bool {

		m1Members, err := m1.Members()
		require.NoError(t, err)
		return len(m1Members) == 2

	}, 10*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {

		m2Members, err := m2.Members()
		require.NoError(t, err)
		return len(m2Members) == 2

	}, 10*time.Second, 100*time.Millisecond)

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
					AdvertiseAddr: "127.0.0.1",
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

				peerAddrs = append(peerAddrs, m.Self().Addr)
			}

			// Check if each node (except the first) has joined
			for i := 1; i < clusterSize; i++ {
				nodeID := nodeIDs[i]
				require.Eventually(t, func() bool {
					joinMu.RLock()
					hasJoined := joinedNodes[nodeID]
					joinMu.RUnlock()
					return hasJoined
				}, 30*time.Second, 100*time.Millisecond)
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
				}, 30*time.Second, 100*time.Millisecond)
			}
		})
	}
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
				AdvertiseAddr: "127.0.0.1",
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
		peerAddrs = append(peerAddrs, m.Self().Addr)
	}

	for _, m := range members {
		require.Eventually(t, func() bool {
			members, err := m.Members()
			require.NoError(t, err)
			return len(members) == n
		}, 5*time.Second, 100*time.Millisecond)
	}
	return members

}
