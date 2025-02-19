package membership_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
)

var CLUSTER_SIZES = []int{3, 9, 36}

func TestJoin2Nodes(t *testing.T) {

	freePort := port.New()

	m1, err := membership.New(membership.Config{
		NodeId:   "node_1",
		SerfAddr: fmt.Sprintf("localhost:%d", freePort.Get()),
		RpcAddr:  fmt.Sprintf("http://localhost:%d", freePort.Get()),
		Logger:   logging.New(nil),
	})
	require.NoError(t, err)

	m2, err := membership.New(membership.Config{
		NodeId:   "node_2",
		SerfAddr: fmt.Sprintf("localhost:%d", freePort.Get()),
		RpcAddr:  fmt.Sprintf("http://localhost:%d", freePort.Get()),
		Logger:   logging.New(nil),
	})
	require.NoError(t, err)

	members, err := m1.Join()
	require.NoError(t, err)
	require.Equal(t, 1, members)

	_, err = m2.Join(m1.SerfAddr())
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		members, err := m2.Members()
		require.NoError(t, err)
		return len(members) == 2

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
			var err error
			for i := 0; i < clusterSize; i++ {
				members[i], err = membership.New(membership.Config{
					NodeId:   fmt.Sprintf("node_%d", i),
					SerfAddr: fmt.Sprintf("localhost:%d", freePort.Get()),
					RpcAddr:  fmt.Sprintf("http://localhost:%d", freePort.Get()),
					Logger:   logging.New(nil),
				})
				require.NoError(t, err)

			}

			joinEvents := members[0].SubscribeJoinEvents()
			joinMu := sync.RWMutex{}
			join := make(map[string]bool)

			go func() {
				for event := range joinEvents {
					joinMu.Lock()
					join[event.NodeId] = true
					joinMu.Unlock()
				}
			}()

			serfAddrs := make([]string, 0)
			for _, m := range members {
				_, err = m.Join(serfAddrs...)
				require.NoError(t, err)
				serfAddrs = append(serfAddrs, m.SerfAddr())
			}

			for _, n := range members[1:] {
				require.Eventually(t, func() bool {
					joinMu.RLock()
					ok := join[n.NodeId()]
					joinMu.RUnlock()
					return ok
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

			leaveEvents := nodes[0].SubscribeLeaveEvents()
			leftMu := sync.RWMutex{}
			left := make(map[string]bool)

			go func() {
				for event := range leaveEvents {
					leftMu.Lock()
					left[event.NodeId] = true
					leftMu.Unlock()
				}
			}()

			for _, n := range nodes[1:] {
				err := n.Leave()
				require.NoError(t, err)
			}

			for _, n := range nodes[1:] {
				require.Eventually(t, func() bool {
					leftMu.RLock()
					l := left[n.NodeId()]
					leftMu.RUnlock()
					return l
				}, 30*time.Second, 100*time.Millisecond)
			}
		})
	}
}

func runMany(t *testing.T, n int) []membership.Membership {

	freePort := port.New()

	members := make([]membership.Membership, n)
	var err error
	for i := 0; i < n; i++ {
		members[i], err = membership.New(membership.Config{
			NodeId:   fmt.Sprintf("node_%d", i),
			SerfAddr: fmt.Sprintf("localhost:%d", freePort.Get()),
			RpcAddr:  fmt.Sprintf("http://localhost:%d", freePort.Get()),
			Logger:   logging.New(nil),
		})
		require.NoError(t, err)

	}

	serfAddrs := make([]string, 0)
	for _, m := range members {
		_, err = m.Join(serfAddrs...)
		require.NoError(t, err)
		serfAddrs = append(serfAddrs, m.SerfAddr())
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
