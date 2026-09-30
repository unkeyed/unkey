package deploy

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubConnection struct {
	net.Conn
}

// TestRegistryDialerRejectsNonPublicDestinations guarantees untrusted registry
// names cannot connect to local infrastructure or special-use addresses.
func TestRegistryDialerRejectsNonPublicDestinations(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		addresses []netip.Addr
	}{
		{name: "loopback", host: "registry.example", addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
		{name: "private", host: "registry.example", addresses: []netip.Addr{netip.MustParseAddr("10.0.0.1")}},
		{name: "link local", host: "registry.example", addresses: []netip.Addr{netip.MustParseAddr("169.254.169.254")}},
		{name: "carrier grade NAT", host: "registry.example", addresses: []netip.Addr{netip.MustParseAddr("100.64.0.1")}},
		{name: "IPv6 private", host: "registry.example", addresses: []netip.Addr{netip.MustParseAddr("fd00::1")}},
		{name: "cluster local", host: "registry.default.svc.cluster.local"},
		{name: "single label", host: "registry"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dialed := false
			dialer := &registryDialer{
				trustedAddresses: map[string]struct{}{},
				lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
					return test.addresses, nil
				},
				dialContext: func(context.Context, string, string) (net.Conn, error) {
					dialed = true
					return &stubConnection{}, nil
				},
			}

			_, err := dialer.DialContext(context.Background(), "tcp", net.JoinHostPort(test.host, "443"))
			require.Error(t, err)
			require.False(t, dialed, "blocked destinations must never reach the network dialer")
		})
	}
}

// TestRegistryDialerConnectsOnlyToResolvedPublicAddress guarantees the dialer
// connects to the checked IP rather than resolving the hostname a second time.
func TestRegistryDialerConnectsOnlyToResolvedPublicAddress(t *testing.T) {
	var dialedAddress string
	dialer := &registryDialer{
		trustedAddresses: map[string]struct{}{},
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{
				netip.MustParseAddr("10.0.0.1"),
				netip.MustParseAddr("8.8.8.8"),
			}, nil
		},
		dialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialedAddress = address
			return &stubConnection{}, nil
		},
	}

	_, err := dialer.DialContext(context.Background(), "tcp", "registry.example:443")
	require.NoError(t, err)
	require.Equal(t, "8.8.8.8:443", dialedAddress)
}

// TestRegistryDialerAllowsConfiguredInternalRegistry guarantees Unkey's trusted
// build registry remains reachable when it uses internal DNS.
func TestRegistryDialerAllowsConfiguredInternalRegistry(t *testing.T) {
	var dialedAddress string
	dialer := &registryDialer{
		trustedAddresses: map[string]struct{}{
			"registry.internal:443": {},
		},
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			t.Fatal("trusted registry must not use the public-only resolver")
			return nil, nil
		},
		dialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialedAddress = address
			return &stubConnection{}, nil
		},
	}

	_, err := dialer.DialContext(context.Background(), "tcp", "registry.internal:443")
	require.NoError(t, err)
	require.Equal(t, "registry.internal:443", dialedAddress)
}

// TestRegistryDialerRejectsSpecialIPv6Destinations guarantees transition and
// translation ranges cannot hide a non-public destination.
func TestRegistryDialerRejectsSpecialIPv6Destinations(t *testing.T) {
	addresses := []netip.Addr{
		netip.MustParseAddr("fec0::1"),
		netip.MustParseAddr("::7f00:1"),
		netip.MustParseAddr("64:ff9b::7f00:1"),
		netip.MustParseAddr("2002:7f00:1::"),
	}
	for _, address := range addresses {
		require.False(t, isPublicRegistryAddress(address), address.String())
	}
	require.False(t, isPublicRegistryAddress(netip.MustParseAddr("192.88.99.1")))
}
