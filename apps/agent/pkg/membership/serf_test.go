package membership_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
)

var freePort *port.FreePort

func TestMembership(t *testing.T) {
	freePort = port.New()
	m1 := setupMember(t)
	m2 := setupMember(t, m1)
	m3 := setupMember(t, m1, m2)

	t.Cleanup(func() {
		require.NoError(t, m1.Shutdown())
		require.NoError(t, m2.Shutdown())
		require.NoError(t, m3.Shutdown())
	})

	// all nodes should see 3 members
	require.Eventually(t, func() bool {
		return 3 == len(m1.Members()) && 3 == len(m2.Members()) && 3 == len(m3.Members())
	}, 3*time.Second, 250*time.Millisecond)

	require.NoError(t, m2.Leave())

	// m1 should see 2 members
	require.Eventually(t, func() bool {
		return len(m1.Members()) == 2
	}, 3*time.Second, 250*time.Millisecond)

	// m3 should see 2 members
	require.Eventually(t, func() bool {
		return len(m3.Members()) == 2
	}, 3*time.Second, 250*time.Millisecond)

	m4 := setupMember(t, m3)
	t.Cleanup(func() {
		require.NoError(t, m4.Shutdown())
	})

	// m1 should see 3 members
	require.Eventually(t, func() bool {
		return len(m1.Members()) == 3
	}, 3*time.Second, 250*time.Millisecond)

	// m3 should see 3 members
	require.Eventually(t, func() bool {
		return len(m3.Members()) == 3
	}, 3*time.Second, 250*time.Millisecond)

	// m4 should see 3 members
	require.Eventually(t, func() bool {
		return len(m4.Members()) == 3
	}, 3*time.Second, 250*time.Millisecond)

}

func setupMember(t *testing.T, members ...*membership.Membership) *membership.Membership {
	id := uuid.New().String()

	c := membership.Config{
		NodeId:   id,
		SerfAddr: fmt.Sprintf("%s:%d", "localhost", freePort.Get()),
		RpcAddr:  fmt.Sprintf("%s:%d", "localhost", freePort.Get()),
		Region:   "test",
		Logger:   logging.NewNoopLogger(),
	}

	n, err := membership.New(c)
	require.NoError(t, err)

	addrs := make([]string, len(members))
	for i, m := range members {
		addrs[i] = m.SerfAddr()
	}
	err = n.Join(addrs...)
	require.NoError(t, err)
	return n

}
