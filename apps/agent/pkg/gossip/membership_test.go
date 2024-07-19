package gossip

import (
	"context"
	"fmt"
	"sync"

	// "sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
)

var CLUSTER_SIZES = []int{3, 9, 36}

func TestJoin2Nodes(t *testing.T) {

	freePort := port.New()

	m1, err := New(Config{
		NodeId:  "node_1",
		RpcAddr: fmt.Sprintf("http://localhost:%d", freePort.Get()),
		Logger:  logging.New(&logging.Config{Debug: true}),
	})
	require.NoError(t, err)

	require.NoError(t, err)

	go NewClusterServer(m1, logging.New(&logging.Config{Debug: true})).Serve()

	m2, err := New(Config{
		NodeId:  "node_2",
		RpcAddr: fmt.Sprintf("http://localhost:%d", freePort.Get()),
		Logger:  logging.New(&logging.Config{Debug: true}),
	})
	require.NoError(t, err)

	go NewClusterServer(m2, logging.New(&logging.Config{Debug: true})).Serve()

	t.Logf("m1 addr: %s", m1.RpcAddr())

	err = m2.Join(context.Background(), m1.RpcAddr())
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		members := m2.Members()
		require.NoError(t, err)
		return len(members) == 2

	}, 10*time.Second, 100*time.Millisecond)

}

func TestMembers_returns_all_members(t *testing.T) {

	for _, clusterSize := range CLUSTER_SIZES {

		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {

			nodes := runMany(t, clusterSize)
			for _, m := range nodes {
				require.Eventually(t, func() bool {
					return len(m.Members()) == clusterSize
				}, time.Minute, 100*time.Millisecond)

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
			members := make([]*cluster, clusterSize)
			var err error
			for i := 0; i < clusterSize; i++ {
				members[i], err = New(Config{
					NodeId:  fmt.Sprintf("node_%d", i),
					RpcAddr: fmt.Sprintf("http://localhost:%d", freePort.Get()),
					Logger:  logging.New(&logging.Config{Debug: true}),
				})
				require.NoError(t, err)
				go NewClusterServer(members[i], logging.New(&logging.Config{Debug: true})).Serve()
			}

			joinEvents := members[0].SubscribeJoinEvents("test")
			joinMu := sync.RWMutex{}
			join := make(map[string]bool)

			go func() {
				for event := range joinEvents {
					joinMu.Lock()
					join[event.NodeId] = true
					joinMu.Unlock()
				}
			}()

			rpcAddrs := make([]string, 0)
			for _, m := range members {
				err := m.Join(context.Background(), rpcAddrs...)
				require.NoError(t, err)
				rpcAddrs = append(rpcAddrs, m.RpcAddr())
			}

			for _, n := range members[1:] {
				require.Eventually(t, func() bool {
					joinMu.RLock()
					t.Logf("joins: %+v", join)
					ok := join[n.self.NodeId]
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

			leaveEvents := nodes[0].SubscribeLeaveEvents("test")
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
				err := n.Shutdown(context.Background())
				require.NoError(t, err)
			}

			for _, n := range nodes[1:] {
				require.Eventually(t, func() bool {
					leftMu.RLock()
					t.Log(left)
					l := left[n.self.NodeId]
					leftMu.RUnlock()
					return l
				}, 30*time.Second, 100*time.Millisecond)
			}
		})
	}
}

func TestUncleanShutdown(t *testing.T) {
	t.Skip("WIP")
	for _, clusterSize := range CLUSTER_SIZES {

		t.Run(fmt.Sprintf("cluster size %d", clusterSize), func(t *testing.T) {

			freePort := port.New()

			gossipFactor := 3
			if clusterSize < gossipFactor {
				gossipFactor = clusterSize - 1
			}

			members := make([]*cluster, clusterSize)
			srvs := make([]*clusterServer, clusterSize)
			for i := 0; i < clusterSize; i++ {
				c, err := New(Config{
					NodeId:       fmt.Sprintf("node_%d", i),
					RpcAddr:      fmt.Sprintf("http://localhost:%d", freePort.Get()),
					Logger:       logging.New(&logging.Config{Debug: true}),
					GossipFactor: gossipFactor,
				})
				require.NoError(t, err)

				srv := NewClusterServer(c, logging.New(&logging.Config{Debug: true}))
				go srv.Serve()
				members[i] = c
				srvs[i] = srv
			}

			rpcAddrs := make([]string, 0)
			for _, m := range members {
				err := m.Join(context.Background(), rpcAddrs...)
				require.NoError(t, err)
				rpcAddrs = append(rpcAddrs, m.self.RpcAddr)
			}

			for _, m := range members {
				require.Eventually(t, func() bool {
					return len(m.Members()) == clusterSize
				}, 5*time.Second, 100*time.Millisecond)
			}

			srvs[0]._testSimulateFailure()

			for _, m := range members[1:] {
				require.Eventually(t, func() bool {
					t.Logf("members: %+v", m.Members())
					return len(m.Members()) == clusterSize-1
				}, 10*time.Second, 100*time.Millisecond)
			}

		})
	}
}

func runMany(t *testing.T, n int) []*cluster {
	freePort := port.New()

	gossipFactor := 3
	if n < gossipFactor {
		gossipFactor = n - 1
	}

	members := make([]*cluster, n)
	for i := 0; i < n; i++ {
		c, err := New(Config{
			NodeId:       fmt.Sprintf("node_%d", i),
			RpcAddr:      fmt.Sprintf("http://localhost:%d", freePort.Get()),
			Logger:       logging.New(&logging.Config{Debug: true}),
			GossipFactor: gossipFactor,
		})
		require.NoError(t, err)

		members[i] = c

		srv := NewClusterServer(c, logging.New(&logging.Config{Debug: true}))
		go srv.Serve()
	}

	rpcAddrs := make([]string, 0)
	for _, m := range members {
		err := m.Join(context.Background(), rpcAddrs...)
		require.NoError(t, err)
		rpcAddrs = append(rpcAddrs, m.self.RpcAddr)
	}

	for _, m := range members {
		require.Eventually(t, func() bool {
			return len(m.Members()) == n
		}, 5*time.Second, 100*time.Millisecond)
	}
	return members

}
