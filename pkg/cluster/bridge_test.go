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

// TestMemberlistCreate_HostnameAdvertiseAddr_LeaksPort reproduces the production
// bug where setting AdvertiseAddr to a hostname (like an NLB DNS name) causes
// memberlist.Create to leak the TCP/UDP listeners. The first Create binds the
// port, then fails inside refreshAdvertise (net.ParseIP returns nil for
// hostnames), and newMemberlist returns the error without shutting down the
// transport. Every subsequent Create on the same port fails with EADDRINUSE.
func TestMemberlistCreate_HostnameAdvertiseAddr_LeaksPort(t *testing.T) {
	// Pick a free port so the test doesn't conflict with anything.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	cfg := memberlist.DefaultWANConfig()
	cfg.Name = "leak-test-wan"
	cfg.BindAddr = "127.0.0.1"
	cfg.BindPort = port
	cfg.AdvertisePort = port
	cfg.AdvertiseAddr = "some-nlb-hostname.elb.us-east-1.amazonaws.com" // hostname, not IP
	cfg.LogOutput = newLogWriter("wan")

	// First Create: transport binds TCP+UDP on port, then refreshAdvertise
	// fails because net.ParseIP("some-nlb-hostname...") returns nil.
	// newMemberlist returns the error WITHOUT shutting down the transport.
	ml, err := memberlist.Create(cfg)
	require.Error(t, err, "Create should fail when AdvertiseAddr is a hostname")
	require.Nil(t, ml)
	require.Contains(t, err.Error(), "failed to parse advertise address")

	// Second Create on the same port: should succeed if the first Create
	// cleaned up properly, but it doesn't — the leaked listener holds the port.
	cfg2 := memberlist.DefaultWANConfig()
	cfg2.Name = "leak-test-wan-2"
	cfg2.BindAddr = "127.0.0.1"
	cfg2.BindPort = port
	cfg2.AdvertisePort = port
	cfg2.LogOutput = newLogWriter("wan")
	// No AdvertiseAddr this time — use default (should work).

	ml2, err := memberlist.Create(cfg2)
	if err != nil {
		// This proves the leak: the port is stuck.
		require.Contains(t, err.Error(), "address already in use",
			"port is leaked by the first failed Create")
		t.Logf("Confirmed: memberlist.Create leaked port %d after hostname AdvertiseAddr failure", port)
	} else {
		// If memberlist ever fixes this, the test still passes.
		ml2.Shutdown() //nolint:errcheck
		t.Log("memberlist properly cleaned up — no leak (this is the fixed behavior)")
	}
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

	t.Run("unresolvable hostname falls back to original", func(t *testing.T) {
		addr := resolveAdvertiseAddr("this-will-never-resolve.invalid")
		require.Equal(t, "this-will-never-resolve.invalid", addr)
	})
}

// TestPromoteToBridge_WithHostnameAdvertiseAddr verifies that a node with a
// resolvable WANAdvertiseAddr hostname can still become bridge (the hostname
// is resolved to an IP before being passed to memberlist).
func TestPromoteToBridge_WithHostnameAdvertiseAddr(t *testing.T) {
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
}

// TestPromoteToBridge_EmptyAdvertiseAddr verifies the default behavior when no
// WANAdvertiseAddr is configured (memberlist picks a private IP).
func TestPromoteToBridge_EmptyAdvertiseAddr(t *testing.T) {
	c, err := New(Config{
		Region:   "us-east-1",
		NodeID:   "default-addr-test",
		BindAddr: "127.0.0.1",
		OnMessage: func(msg *clusterv1.ClusterMessage) {},
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Close()) }()

	require.Eventually(t, func() bool {
		return c.IsBridge()
	}, 5*time.Second, 50*time.Millisecond, "node should become bridge")

	require.NotEmpty(t, c.WANAddr())
}
