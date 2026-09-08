package ssrf

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsForbiddenIP guarantees the SSRF guard rejects every address class
// that reaches our own infrastructure (loopback, RFC 1918, link-local
// including the cloud metadata range, ULA) while allowing public addresses.
func TestIsForbiddenIP(t *testing.T) {
	tests := map[string]bool{
		"127.0.0.1": true, "10.0.0.1": true, "172.16.0.1": true,
		"192.168.1.1": true, "169.254.169.254": true, "::1": true,
		"fe80::1": true, "fc00::1": true, "100.64.0.1": true,
		"64:ff9b::7f00:1": true, "93.184.216.34": false,
		"2606:2800:220:1:248:1893:25c8:1946": false,
	}
	for raw, forbidden := range tests {
		t.Run(raw, func(t *testing.T) {
			require.Equal(t, forbidden, IsForbiddenIP(net.ParseIP(raw)))
		})
	}
}

// TestIsForbiddenAddr guarantees netip callers get the same private-address
// protection, including IPv4 addresses represented as IPv6.
func TestIsForbiddenAddr(t *testing.T) {
	require.True(t, IsForbiddenAddr(netip.Addr{}))
	require.True(t, IsForbiddenAddr(netip.MustParseAddr("::ffff:127.0.0.1")))
	require.False(t, IsForbiddenAddr(netip.MustParseAddr("93.184.216.34")))
}

// TestRedirectsAreNotFollowedByDefault guarantees that a redirecting endpoint
// cannot bounce a client from [New] to another destination unless the caller
// opted in with [FollowRedirects].
func TestRedirectsAreNotFollowedByDefault(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirecting.Close)

	// Servers listen on loopback over plain http, so the guards must be off.
	client := New(UnsafeAllowAll())
	//nolint:noctx // Direct Get keeps the redirect-policy test minimal.
	resp, err := client.Get(redirecting.URL)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}
	require.ErrorContains(t, err, "redirects are not allowed")

	client = New(UnsafeAllowAll(), FollowRedirects(3))
	//nolint:noctx // Direct Get keeps the redirect-policy test minimal.
	resp, err = client.Get(redirecting.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestValidateEndpoint guarantees that configuration-time validation only
// accepts absolute https URLs, with plain http permitted solely behind
// [UnsafeAllowAll].
func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		opts     []Option
		wantErr  bool
	}{
		{name: "accepts https", endpoint: "https://example.com/logs", wantErr: false},
		{name: "rejects http", endpoint: "http://example.com/logs", wantErr: true},
		{name: "accepts http with UnsafeAllowAll", endpoint: "http://example.com/logs", opts: []Option{UnsafeAllowAll()}, wantErr: false},
		{name: "rejects relative URL", endpoint: "example.com/logs", wantErr: true},
		{name: "rejects empty string", endpoint: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
