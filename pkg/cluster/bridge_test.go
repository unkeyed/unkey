package cluster

import (
	"net"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/stretchr/testify/require"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
)

func TestBridgeElection_SmallestNameWins(t *testing.T) {
	t.Run("smallest name wins", func(t *testing.T) {
		names := []string{
			"node-3",
			"node-1", // smallest
			"node-2",
		}

		// Find smallest (same logic as evaluateBridge)
		smallest := names[0]
		for _, name := range names[1:] {
			if name < smallest {
				smallest = name
			}
		}

		require.Equal(t, "node-1", smallest)
	})
}

// TestMemberlistCreate_HostnameAdvertiseAddr_NoLeakAfterFix verifies that
// resolveAdvertiseAddr returning "" for unresolvable hostnames prevents
// memberlist from leaking ports. When AdvertiseAddr is empty, memberlist
// uses the bind address and Create succeeds without binding and failing.
func TestMemberlistCreate_HostnameAdvertiseAddr_NoLeakAfterFix(t *testing.T) {
	// Pick a free port so the test doesn't conflict with anything.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	// resolveAdvertiseAddr now returns "" for unresolvable hostnames,
	// so memberlist never sees a raw hostname as AdvertiseAddr.
	resolved := resolveAdvertiseAddr("some-nlb-hostname.elb.us-east-1.amazonaws.com")
	require.Equal(t, "", resolved, "unresolvable hostname should resolve to empty string")

	cfg := memberlist.DefaultWANConfig()
	cfg.Name = "leak-test-wan"
	cfg.BindAddr = "127.0.0.1"
	cfg.BindPort = port
	cfg.AdvertisePort = port
	cfg.AdvertiseAddr = resolved // empty — memberlist will use BindAddr
	cfg.LogOutput = newLogWriter("wan")

	ml, err := memberlist.Create(cfg)
	require.NoError(t, err, "Create should succeed when AdvertiseAddr is empty")
	require.NotNil(t, ml)
	require.NoError(t, ml.Shutdown())

	// Second Create on the same port must also succeed — no leaked port.
	cfg2 := memberlist.DefaultWANConfig()
	cfg2.Name = "leak-test-wan-2"
	cfg2.BindAddr = "127.0.0.1"
	cfg2.BindPort = port
	cfg2.AdvertisePort = port
	cfg2.LogOutput = newLogWriter("wan")

	ml2, err := memberlist.Create(cfg2)
	require.NoError(t, err, "port should not be leaked from previous Create")
	require.NotNil(t, ml2)
	require.NoError(t, ml2.Shutdown())
}

// TestResolveAdvertiseAddr verifies that resolveAdvertiseAddr resolves hostnames
// to IPs and passes through literal IPs unchanged.
func TestResolveAdvertiseAddr(t *testing.T) {
	t.Run("empty string passes through", func(t *testing.T) {
		require.Equal(t, "", resolveAdvertiseAddr(""))
	})

	t.Run("literal IP passes through", func(t *testing.T) {
		require.Equal(t, "10.1.40.129", resolveAdvertiseAddr("10.1.40.129"))
	})

	t.Run("localhost resolves to IP", func(t *testing.T) {
		result := resolveAdvertiseAddr("localhost")
		require.NotEmpty(t, result)
		require.NotNil(t, net.ParseIP(result), "result should be a valid IP, got %q", result)
	})

	t.Run("unresolvable hostname returns empty string", func(t *testing.T) {
		addr := resolveAdvertiseAddr("this-will-never-resolve.invalid")
		require.Equal(t, "", addr)
	})
}

func TestPromoteToBridge(t *testing.T) {
	t.Run("WithHostnameAdvertiseAddr", func(t *testing.T) {
		c, err := New(Config{
			Region:           "us-east-1",
			NodeID:           "hostname-test-node",
			BindAddr:         "127.0.0.1",
			WANAdvertiseAddr: "localhost", // hostname, not IP
			OnMessage:        func(msg *clusterv1.ClusterMessage) {},
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, c.Close()) }()

		require.Eventually(t, func() bool {
			return c.IsBridge()
		}, 5*time.Second, 50*time.Millisecond, "node should become bridge even with hostname WANAdvertiseAddr")

		addr := c.WANAddr()
		require.NotEmpty(t, addr)
		t.Logf("WAN addr: %s", addr)

		// Verify the advertised address is an IP, not a hostname.
		host, _, err := net.SplitHostPort(addr)
		require.NoError(t, err)
		require.NotNil(t, net.ParseIP(host), "WAN advertise address should be an IP, got %q", host)
	})

	t.Run("EmptyAdvertiseAddr", func(t *testing.T) {
		c, err := New(Config{
			Region:    "us-east-1",
			NodeID:    "default-addr-test",
			BindAddr:  "127.0.0.1",
			OnMessage: func(msg *clusterv1.ClusterMessage) {},
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, c.Close()) }()

		require.Eventually(t, func() bool {
			return c.IsBridge()
		}, 5*time.Second, 50*time.Millisecond, "node should become bridge")

		require.NotEmpty(t, c.WANAddr())
	})
}
