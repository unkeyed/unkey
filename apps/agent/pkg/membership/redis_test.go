package membership_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutils/containers"
)

func TestJoin2Nodes(t *testing.T) {
	redis := containers.NewRedis(t)
	defer redis.Stop()

	freePort := port.New()

	m1, err := membership.New(membership.Config{
		RedisUrl: redis.URL,
		NodeId:   "node_1",
		RpcAddr:  fmt.Sprintf("http://localhost:%d", freePort.Get()),
	})
	require.NoError(t, err)

	m2, err := membership.New(membership.Config{
		RedisUrl: redis.URL,
		NodeId:   "node_2",
		RpcAddr:  fmt.Sprintf("http://localhost:%d", freePort.Get()),
	})
	require.NoError(t, err)

	members, err := m1.Join()
	require.NoError(t, err)
	require.Equal(t, 1, members)

	members, err = m2.Join()
	require.NoError(t, err)
	require.Equal(t, 2, members)

}

func TestMembers_returns_all_members(t *testing.T) {
	redis := containers.NewRedis(t)
	defer redis.Stop()

	const NUMBER_OF_NODES = 9

	nodes := runMany(t, redis.URL, NUMBER_OF_NODES)

	for _, m := range nodes {
		members, err := m.Members()
		require.NoError(t, err)
		require.Len(t, members, NUMBER_OF_NODES)

	}

}

// Test whether nodes correctly emit a leave event when they leave the cluster
// When a node leaves, a listener to the `LeaveEvents` topic should receive a message with the member that left.
func TestLeave_emits_leave_event(t *testing.T) {
	redis := containers.NewRedis(t)
	defer redis.Stop()

	const NUMBER_OF_NODES = 9

	nodes := runMany(t, redis.URL, NUMBER_OF_NODES)

	leaveEvents := nodes[0].SubscribeLeaveEvents()
	leftMu := sync.RWMutex{}
	left := make(map[string]bool)

	go func() {
		for event := range leaveEvents {
			leftMu.Lock()
			left[event.Id] = true
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

}

func runMany(t *testing.T, redisURL string, n int) []membership.Membership {

	// freePort := port.New()

	members := make([]membership.Membership, n)
	var err error
	for i := 0; i < n; i++ {
		members[i], err = membership.New(membership.Config{
			RedisUrl:      redisURL,
			NodeId:        fmt.Sprintf("node_%d", i),
			RpcAddr:       fmt.Sprintf("http://localhost:%d", 23456+i),
			SyncTtl:       2 * time.Second,
			SyncFrequency: 1 * time.Second,
		})
		require.NoError(t, err)

	}

	for _, m := range members {
		_, err = m.Join()
		require.NoError(t, err)
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
