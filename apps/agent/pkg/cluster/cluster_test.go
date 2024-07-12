package cluster

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
)

func TestMembershipChangesArePropagatedToHashRing(t *testing.T) {

	c1 := createCluster(t, "node1")
	contacted, err := c1.membership.Join()
	require.NoError(t, err)
	require.Equal(t, 1, contacted)

	// Create a 2nd node
	c2 := createCluster(t, "node2")
	contacted, err = c2.membership.Join(c1.membership.SerfAddr())
	require.NoError(t, err)
	require.Equal(t, 2, contacted)

	// Check if the hash rings are updated
	require.Eventually(t, func() bool {

		return len(c1.ring.Members()) == 2
	}, time.Minute, time.Second)

	require.Eventually(t, func() bool {
		return len(c2.ring.Members()) == 2
	}, time.Minute, time.Second)

	// If we shut down one node now, the other one should eventually reduce it's ring size to 1
	err = c2.Shutdown()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(c1.ring.Members()) == 1
	}, time.Minute, time.Second)
}

func createCluster(t *testing.T, nodeId string) *cluster {
	t.Helper()

	logger := logging.New(nil).With().Str("nodeId", nodeId).Logger()

	rpcAddr := fmt.Sprintf("localhost:%d", port.Get())

	m, err := membership.New(membership.Config{
		NodeId:   nodeId,
		Logger:   logger,
		SerfAddr: fmt.Sprintf("localhost:%d", port.Get()),
		RpcAddr:  rpcAddr,
	})
	require.NoError(t, err)

	c, err := New(Config{
		NodeId:     nodeId,
		Membership: m,
		Logger:     logger,
		Debug:      true,
		RpcAddr:    rpcAddr,
		AuthToken:  "test-auth-token",
	})
	require.NoError(t, err)

	// we need to look into the internals of the cluster to access to the hashring which is usually
	// hidden behind the interface
	clusterAsStruct, ok := c.(*cluster)
	require.True(t, ok)
	return clusterAsStruct

}
