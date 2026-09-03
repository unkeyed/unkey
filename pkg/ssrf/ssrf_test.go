package ssrf

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestIsForbiddenIP guarantees the SSRF guard rejects non-public and IANA
// special-purpose addresses, including IPv4 addresses in IPv6-mapped form.
func TestIsForbiddenIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        net.IP
		forbidden bool
	}{
		{name: "nil", ip: net.IP(nil), forbidden: true},
		{name: "IPv4 unspecified", ip: net.ParseIP("0.0.0.0"), forbidden: true},
		{name: "IPv6 unspecified", ip: net.ParseIP("::"), forbidden: true},
		{name: "IPv4 loopback", ip: net.ParseIP("127.0.0.1"), forbidden: true},
		{name: "IPv4 private 10", ip: net.ParseIP("10.0.0.1"), forbidden: true},
		{name: "IPv4 private 172", ip: net.ParseIP("172.16.0.1"), forbidden: true},
		{name: "IPv4 private 192", ip: net.ParseIP("192.168.1.1"), forbidden: true},
		{name: "cloud metadata", ip: net.ParseIP("169.254.169.254"), forbidden: true},
		{name: "IPv4 multicast lower", ip: net.ParseIP("224.0.0.1"), forbidden: true},
		{name: "IPv4 multicast upper", ip: net.ParseIP("239.255.255.250"), forbidden: true},
		{name: "IPv6 loopback", ip: net.ParseIP("::1"), forbidden: true},
		{name: "IPv6 link local", ip: net.ParseIP("fe80::1"), forbidden: true},
		{name: "IPv6 private", ip: net.ParseIP("fc00::1"), forbidden: true},
		{name: "IPv6 link-local multicast", ip: net.ParseIP("ff02::1"), forbidden: true},
		{name: "IPv6 site-local multicast", ip: net.ParseIP("ff05::1"), forbidden: true},
		{name: "mapped loopback", ip: net.ParseIP("::ffff:127.0.0.1"), forbidden: true},
		{name: "mapped private", ip: net.ParseIP("::ffff:10.0.0.1"), forbidden: true},
		{name: "shared address lower", ip: net.ParseIP("100.64.0.1"), forbidden: true},
		{name: "Alibaba metadata", ip: net.ParseIP("100.100.100.200"), forbidden: true},
		{name: "shared address upper", ip: net.ParseIP("100.127.255.255"), forbidden: true},
		{name: "outside shared address", ip: net.ParseIP("100.128.0.1"), forbidden: false},
		{name: "IETF protocol assignment", ip: net.ParseIP("192.0.0.1"), forbidden: true},
		{name: "outside IETF protocol assignments", ip: net.ParseIP("192.0.1.1"), forbidden: false},
		{name: "benchmarking lower", ip: net.ParseIP("198.18.0.1"), forbidden: true},
		{name: "benchmarking upper", ip: net.ParseIP("198.19.255.255"), forbidden: true},
		{name: "outside benchmarking", ip: net.ParseIP("198.20.0.1"), forbidden: false},
		{name: "limited broadcast", ip: net.ParseIP("255.255.255.255"), forbidden: true},
		{name: "NAT64 loopback", ip: net.ParseIP("64:ff9b::7f00:1"), forbidden: true},
		// Translators can forward the embedded public IPv4 address, so the entire NAT64 prefix is forbidden.
		{name: "NAT64 public IPv4", ip: net.ParseIP("64:ff9b::808:808"), forbidden: true},
		{name: "outside NAT64", ip: net.ParseIP("64:ff9b:1::1"), forbidden: false},
		{name: "6to4 loopback", ip: net.ParseIP("2002:7f00:1::"), forbidden: true},
		{name: "6to4 public IPv4", ip: net.ParseIP("2002:808:808::"), forbidden: true},
		{name: "outside 6to4", ip: net.ParseIP("2003::1"), forbidden: false},
		{name: "mapped shared address", ip: net.ParseIP("::ffff:100.64.0.1"), forbidden: true},
		{name: "mapped public", ip: net.ParseIP("::ffff:8.8.8.8"), forbidden: false},
		{name: "Google DNS", ip: net.ParseIP("8.8.8.8"), forbidden: false},
		{name: "Cloudflare DNS", ip: net.ParseIP("1.1.1.1"), forbidden: false},
		{name: "public IPv4", ip: net.ParseIP("93.184.216.34"), forbidden: false},
		{name: "public IPv6", ip: net.ParseIP("2606:2800:220:1:248:1893:25c8:1946"), forbidden: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.forbidden, IsForbiddenIP(tt.ip))
		})
	}

	t.Run("rejects malformed IP", func(t *testing.T) {
		require.True(t, IsForbiddenIP(net.IP{1, 2, 3}))
	})
}

// TestNewBlocksForbiddenResolvedIP guarantees the hardened client rejects a
// real connection to loopback before reaching the listening server.
func TestNewBlocksForbiddenResolvedIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := New().Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}
	require.ErrorContains(t, err, "resolved only to forbidden IP addresses")
}

// TestDialContext guarantees DNS failures, forbidden answers, and connection
// fallback are enforced against the exact resolved addresses being dialed.
func TestDialContext(t *testing.T) {
	t.Run("rejects all forbidden answers promptly", func(t *testing.T) {
		var resolvedHost string
		cfg := applyOptions(nil)
		cfg.lookupIPAddr = func(_ context.Context, host string) ([]net.IPAddr, error) {
			resolvedHost = host
			return []net.IPAddr{{IP: net.ParseIP("10.0.0.1")}, {IP: net.ParseIP("192.168.0.1")}}, nil
		}

		started := time.Now()
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		t.Cleanup(cancel)
		conn, err := dialContext(cfg)(ctx, "tcp", "example.test:80")
		require.Nil(t, conn)
		require.ErrorContains(t, err, "resolved only to forbidden IP addresses")
		require.Equal(t, "example.test", resolvedHost)
		require.Less(t, time.Since(started), 5*time.Second)
	})

	t.Run("allows loopback only when guards are disabled", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, listener.Close()) })
		require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(2*time.Second)))
		port := listener.Addr().(*net.TCPAddr).Port

		cfg := applyOptions([]Option{UnsafeAllowAll()})
		cfg.lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
		}
		conn, err := dialContext(cfg)(t.Context(), "tcp", net.JoinHostPort("example.test", strconv.Itoa(port)))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, conn.Close()) })
		accepted, err := listener.Accept()
		require.NoError(t, err)
		require.NoError(t, accepted.Close())
	})

	t.Run("returns dial error after allowed answer fails", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		address := listener.Addr().String()
		require.NoError(t, listener.Close())
		cfg := applyOptions([]Option{UnsafeAllowAll()})
		cfg.lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
		}

		conn, err := dialContext(cfg)(t.Context(), "tcp", address)
		require.Nil(t, conn)
		require.ErrorContains(t, err, "dial endpoint:")
	})

	t.Run("tries the next answer after a dial failure", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, listener.Close()) })
		require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(2*time.Second)))
		port := listener.Addr().(*net.TCPAddr).Port
		cfg := applyOptions([]Option{UnsafeAllowAll()})
		cfg.lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("127.0.0.2")}, {IP: net.ParseIP("127.0.0.1")}}, nil
		}

		conn, err := dialContext(cfg)(t.Context(), "tcp", net.JoinHostPort("example.test", strconv.Itoa(port)))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, conn.Close()) })
		accepted, err := listener.Accept()
		require.NoError(t, err)
		require.NoError(t, accepted.Close())
	})

	t.Run("wraps resolver errors", func(t *testing.T) {
		cfg := applyOptions(nil)
		cfg.lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
			return nil, errors.New("DNS unavailable")
		}
		conn, err := dialContext(cfg)(t.Context(), "tcp", "example.test:443")
		require.Nil(t, conn)
		require.ErrorContains(t, err, "resolve endpoint host:")
	})

	t.Run("rejects an address without a port", func(t *testing.T) {
		conn, err := dialContext(applyOptions(nil))(t.Context(), "tcp", "example.test")
		require.Nil(t, conn)
		require.ErrorContains(t, err, "split dial address:")
	})

	t.Run("rejects mapped loopback", func(t *testing.T) {
		cfg := applyOptions(nil)
		cfg.lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("::ffff:127.0.0.1")}}, nil
		}
		conn, err := dialContext(cfg)(t.Context(), "tcp", "example.test:443")
		require.Nil(t, conn)
		require.ErrorContains(t, err, "resolved only to forbidden IP addresses")
	})
}

// TestCheckRedirect guarantees redirect limits and secure schemes are checked
// independently of the HTTP client's redirect machinery.
func TestCheckRedirect(t *testing.T) {
	httpsURL := &url.URL{Scheme: "https", Host: "example.com"}
	httpURL := &url.URL{Scheme: "http", Host: "example.com"}

	t.Run("rejects redirects by default", func(t *testing.T) {
		err := checkRedirect(applyOptions(nil))(&http.Request{URL: httpsURL}, nil)
		require.ErrorContains(t, err, "redirects are not allowed")
	})
	t.Run("permits the configured number of redirects", func(t *testing.T) {
		check := checkRedirect(applyOptions([]Option{FollowRedirects(3)}))
		require.NoError(t, check(&http.Request{URL: httpsURL}, make([]*http.Request, 1)))
		require.NoError(t, check(&http.Request{URL: httpsURL}, make([]*http.Request, 3)))
		require.ErrorContains(t, check(&http.Request{URL: httpsURL}, make([]*http.Request, 4)), "stopped after 3 redirects")
	})
	t.Run("rejects plain HTTP with guards enabled", func(t *testing.T) {
		err := checkRedirect(applyOptions([]Option{FollowRedirects(3)}))(&http.Request{URL: httpURL}, nil)
		require.ErrorContains(t, err, "redirect endpoint must use https")
	})
	t.Run("permits plain HTTP with guards disabled", func(t *testing.T) {
		err := checkRedirect(applyOptions([]Option{UnsafeAllowAll(), FollowRedirects(3)}))(&http.Request{URL: httpURL}, nil)
		require.NoError(t, err)
	})
}

// TestRedirectLimitEndToEnd guarantees the configured redirect count is
// enforced across an actual chain of HTTP responses.
func TestRedirectLimitEndToEnd(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(second.Close)
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, second.URL, http.StatusFound)
	}))
	t.Cleanup(first.Close)

	request := func() *http.Request {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, first.URL, nil)
		require.NoError(t, err)
		return req
	}
	resp, err := New(UnsafeAllowAll(), FollowRedirects(1)).Do(request())
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}
	require.ErrorContains(t, err, "stopped after 1 redirects")

	resp, err = New(UnsafeAllowAll(), FollowRedirects(2)).Do(request())
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestRedirectsAreNotFollowedByDefault guarantees clients cannot bounce to a
// second destination unless the caller explicitly enables redirects.
func TestRedirectsAreNotFollowedByDefault(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirecting.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, redirecting.URL, nil)
	require.NoError(t, err)
	resp, err := New(UnsafeAllowAll()).Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}
	require.ErrorContains(t, err, "redirects are not allowed")
}

// TestValidateEndpoint guarantees endpoint validation rejects malformed or
// credential-bearing URLs before they can become egress configuration.
func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		opts     []Option
		wantErr  string
	}{
		{name: "accepts https", endpoint: "https://example.com/logs"},
		{name: "accepts https with UnsafeAllowAll", endpoint: "https://example.com/logs", opts: []Option{UnsafeAllowAll()}},
		{name: "rejects http", endpoint: "http://example.com/logs", wantErr: "absolute https URL"},
		{name: "accepts http with UnsafeAllowAll", endpoint: "http://example.com/logs", opts: []Option{UnsafeAllowAll()}},
		{name: "rejects ftp", endpoint: "ftp://example.com/logs", wantErr: "absolute https URL"},
		{name: "rejects relative URL", endpoint: "example.com/logs", wantErr: "absolute https URL"},
		{name: "rejects empty string", endpoint: "", wantErr: "absolute https URL"},
		{name: "rejects empty host", endpoint: "https:///logs", wantErr: "absolute https URL"},
		{name: "rejects unparsable URL", endpoint: "http://[::1", wantErr: "parse endpoint"},
		{name: "rejects userinfo", endpoint: "https://user:pass@example.com/logs", wantErr: "userinfo"},
		{name: "rejects userinfo even with UnsafeAllowAll", endpoint: "http://u:p@example.com", opts: []Option{UnsafeAllowAll()}, wantErr: "userinfo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, tt.opts...)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestNewHardenedTransport guarantees New retains the transport controls that
// prevent requests from bypassing the guarded dial path or consuming excess resources.
func TestNewHardenedTransport(t *testing.T) {
	client := New(WithTimeout(7 * time.Second))
	require.Equal(t, 7*time.Second, client.Timeout)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.DialContext)
	// A proxy would receive the unchecked hostname and bypass the guarded dial path.
	require.Nil(t, transport.Proxy)
	require.False(t, transport.ForceAttemptHTTP2)
	require.Equal(t, 100, transport.MaxIdleConns)
	require.Equal(t, 90*time.Second, transport.IdleConnTimeout)
	require.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
	require.Equal(t, time.Second, transport.ExpectContinueTimeout)
	require.Equal(t, 30*time.Second, transport.ResponseHeaderTimeout)
	require.EqualValues(t, 1<<20, transport.MaxResponseHeaderBytes)

	defaultClient := New()
	require.Zero(t, defaultClient.Timeout)
	require.NotNil(t, defaultClient.CheckRedirect)
}
