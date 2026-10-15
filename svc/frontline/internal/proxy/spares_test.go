package proxy

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type peerFrontline struct {
	server *httptest.Server
	mu     sync.Mutex
	opened map[string]bool
	closed map[string]bool
	abort  map[string]bool
}

// newPeerFrontline serves TLS with HTTP/2 enabled, like a peer frontline, and
// echoes bytes after a WebSocket-style upgrade.
func newPeerFrontline(t *testing.T) *peerFrontline {
	t.Helper()

	//nolint:exhaustruct
	peer := &peerFrontline{opened: map[string]bool{}, closed: map[string]bool{}, abort: map[string]bool{}}
	peer.server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer.mu.Lock()
		abort := peer.abort[r.RemoteAddr]
		peer.mu.Unlock()
		if abort {
			panic(http.ErrAbortHandler)
		}
		if !isUpgradeRequest(r) {
			w.WriteHeader(http.StatusOK)
			return
		}
		conn, rw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
		_ = rw.Flush()
		_, _ = io.Copy(conn, rw)
	}))
	peer.server.EnableHTTP2 = true
	peer.server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		peer.mu.Lock()
		defer peer.mu.Unlock()
		switch state {
		case http.StateNew:
			peer.opened[conn.RemoteAddr().String()] = true
		case http.StateClosed:
			peer.closed[conn.RemoteAddr().String()] = true
		case http.StateActive, http.StateIdle, http.StateHijacked:
		}
	}
	peer.server.StartTLS()
	t.Cleanup(peer.server.Close)
	return peer
}

func (p *peerFrontline) openedAddrs() map[string]bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]bool, len(p.opened))
	for addr := range p.opened {
		out[addr] = true
	}
	return out
}

func (p *peerFrontline) closedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.closed)
}

func newTestSparePool(t *testing.T, peer *peerFrontline, size int, idleAfter time.Duration) *sparePool {
	t.Helper()

	roots := x509.NewCertPool()
	roots.AddCert(peer.server.Certificate())
	//nolint:exhaustruct
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: time.Second},
		Config:    &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}
	pool := newSparePool(dialer, size, idleAfter)
	t.Cleanup(pool.close)
	return pool
}

func requireSpares(t *testing.T, pool *sparePool, n int) {
	t.Helper()
	require.Eventually(t, func() bool {
		pool.mu.Lock()
		defer pool.mu.Unlock()
		for _, peer := range pool.peers {
			return len(peer.conns) == n
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)
}

// upgrade sends a WebSocket-style upgrade, checks that bytes echo over the
// upgraded connection, and returns the local address of the connection used.
func upgrade(t *testing.T, transport http.RoundTripper, url string) string {
	t.Helper()

	var localAddr string
	//nolint:exhaustruct
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { localAddr = info.Conn.LocalAddr().String() },
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(t.Context(), trace), http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	body, ok := resp.Body.(io.ReadWriteCloser)
	require.True(t, ok)
	defer body.Close()
	_, err = body.Write([]byte("ping\n"))
	require.NoError(t, err)
	line, err := bufio.NewReader(body).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ping\n", line)

	return localAddr
}

func TestUpgradeTransport_UsesConnectionDialedBeforeRequest(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	pool := newTestSparePool(t, peer, 2, time.Minute)
	transport := newUpgradeTransport(pool)

	first := upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)

	dialedBefore := peer.openedAddrs()
	second := upgrade(t, transport, peer.server.URL)

	require.NotEqual(t, first, second)
	require.True(t, dialedBefore[second], "second upgrade should use a spare dialed before the request")
}

func TestUpgradeTransport_SkipsSparesClosedByPeer(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	pool := newTestSparePool(t, peer, 2, time.Minute)
	transport := newUpgradeTransport(pool)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)
	stale := peer.openedAddrs()

	peer.server.CloseClientConnections()

	used := upgrade(t, transport, peer.server.URL)
	require.False(t, stale[used], "upgrade used a connection the peer closed")
}

func TestUpgradeTransport_RetriesOnFreshConnectionWhenSpareFails(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	pool := newTestSparePool(t, peer, 2, time.Minute)
	transport := newUpgradeTransport(pool)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)
	spares := peer.openedAddrs()
	peer.mu.Lock()
	peer.abort = spares
	peer.mu.Unlock()

	used := upgrade(t, transport, peer.server.URL)
	require.False(t, spares[used], "upgrade should succeed on a fresh connection")
}

func TestSparePool_ClosesSparesAfterIdle(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	pool := newTestSparePool(t, peer, 2, 200*time.Millisecond)
	transport := newUpgradeTransport(pool)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)

	require.Eventually(t, func() bool { return peer.closedCount() == 2 }, 5*time.Second, 10*time.Millisecond)
	pool.mu.Lock()
	defer pool.mu.Unlock()
	require.Empty(t, pool.peers)
}
