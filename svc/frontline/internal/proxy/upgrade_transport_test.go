package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

type peerFrontline struct {
	server *httptest.Server
	mu     sync.Mutex
	opened map[string]bool
	closed map[string]bool
	abort  map[string]bool
	hang   map[string]bool
}

// newPeerFrontline serves TLS with HTTP/2 enabled, like a peer frontline, and
// echoes bytes after a WebSocket-style upgrade.
func newPeerFrontline(t *testing.T) *peerFrontline {
	t.Helper()

	//nolint:exhaustruct
	peer := &peerFrontline{opened: map[string]bool{}, closed: map[string]bool{}, abort: map[string]bool{}, hang: map[string]bool{}}
	peer.server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer.mu.Lock()
		abort := peer.abort[r.RemoteAddr]
		hang := peer.hang[r.RemoteAddr]
		peer.mu.Unlock()
		if abort {
			panic(http.ErrAbortHandler)
		}
		if hang {
			<-r.Context().Done()
			return
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

func newTestSparePool(t *testing.T, server *httptest.Server, clk clock.Clock, size int, idleAfter time.Duration) *sparePool {
	t.Helper()

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	//nolint:exhaustruct
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: time.Second},
		Config:    &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}
	return newSparePool(dialer, clk, size, idleAfter)
}

var testUpgradeTransportConfig = upgradeTransportConfig{maxIdleConns: 10, idleConnTimeout: time.Minute, spareResponseTimeout: 200 * time.Millisecond}

func requireSpares(t *testing.T, pool *sparePool, n int) {
	t.Helper()
	require.Eventually(t, func() bool {
		pool.mu.Lock()
		defer pool.mu.Unlock()
		for _, peer := range pool.peers {
			return len(peer.spares) == n
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
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, url, nil)
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
	pool := newTestSparePool(t, peer.server, clock.New(), 2, time.Minute)
	transport := newUpgradeTransport(pool, testUpgradeTransportConfig)

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
	pool := newTestSparePool(t, peer.server, clock.New(), 2, time.Minute)
	transport := newUpgradeTransport(pool, testUpgradeTransportConfig)

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
	pool := newTestSparePool(t, peer.server, clock.New(), 2, time.Minute)
	transport := newUpgradeTransport(pool, testUpgradeTransportConfig)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)
	spares := peer.openedAddrs()
	peer.mu.Lock()
	peer.abort = spares
	peer.mu.Unlock()

	used := upgrade(t, transport, peer.server.URL)
	require.False(t, spares[used], "upgrade should succeed on a fresh connection")
}

func TestUpgradeTransport_UpgradeWithBodySkipsSpares(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	pool := newTestSparePool(t, peer.server, clock.New(), 2, time.Minute)
	transport := newUpgradeTransport(pool, testUpgradeTransportConfig)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)
	spares := peer.openedAddrs()
	peer.mu.Lock()
	peer.abort = spares
	peer.mu.Unlock()

	var used string
	//nolint:exhaustruct
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { used = info.Conn.LocalAddr().String() },
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(t.Context(), trace), http.MethodPost, peer.server.URL, strings.NewReader("body"))
	require.NoError(t, err)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err, "an upgrade with a body cannot be retried, so it must not use a spare that may be stale")
	defer resp.Body.Close()
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	require.False(t, spares[used])
}

func TestSparePool_ClosesSparesAfterIdle(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	clk := clock.NewTestClock()
	pool := newTestSparePool(t, peer.server, clk, 2, time.Minute)
	transport := newUpgradeTransport(pool, testUpgradeTransportConfig)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)

	clk.Tick(time.Minute)
	require.Eventually(t, func() bool { return peer.closedCount() == 2 }, 5*time.Second, 10*time.Millisecond)
	pool.mu.Lock()
	defer pool.mu.Unlock()
	require.Empty(t, pool.peers)
}

func TestUpgradeTransport_RetriesOnFreshConnectionWhenSpareNeverAnswers(t *testing.T) {
	t.Parallel()

	peer := newPeerFrontline(t)
	pool := newTestSparePool(t, peer.server, clock.New(), 2, time.Minute)
	transport := newUpgradeTransport(pool, testUpgradeTransportConfig)

	upgrade(t, transport, peer.server.URL)
	requireSpares(t, pool, 2)
	spares := peer.openedAddrs()
	peer.mu.Lock()
	peer.hang = spares
	peer.mu.Unlock()

	used := upgrade(t, transport, peer.server.URL)
	require.False(t, spares[used], "upgrade should succeed on a fresh connection")
}

// newTLSServer starts a TLS server with the given TLS config and a handler
// that does nothing.
func newTLSServer(t *testing.T, cfg *tls.Config) *httptest.Server {
	t.Helper()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.TLS = cfg
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func TestSparePool_DialsForExpiredPeerDoNotJoinNextPeer(t *testing.T) {
	t.Parallel()

	handshakes := make(chan struct{})
	release := sync.OnceFunc(func() { close(handshakes) })
	//nolint:exhaustruct
	server := newTLSServer(t, &tls.Config{
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			<-handshakes
			return nil, nil
		},
	})
	t.Cleanup(release)
	addr := server.Listener.Addr().String()

	clk := clock.NewTestClock()
	pool := newTestSparePool(t, server, clk, 2, time.Minute)
	pool.touchAndTake(addr)
	pool.refill(addr)

	clk.Tick(time.Minute)
	pool.touchAndTake(addr)
	pool.refill(addr)
	release()

	require.Eventually(t, func() bool {
		pool.mu.Lock()
		defer pool.mu.Unlock()
		peer := pool.peers[addr]
		return peer.dialing == 0 && len(peer.spares) == 2
	}, 5*time.Second, 10*time.Millisecond)
}

func TestSparePool_DoesNotRedialSparesClosedRightAfterHandshake(t *testing.T) {
	t.Parallel()

	var handshakes atomic.Int32
	//nolint:exhaustruct
	server := newTLSServer(t, &tls.Config{
		VerifyConnection: func(tls.ConnectionState) error {
			handshakes.Add(1)
			return errors.New("close after handshake")
		},
	})
	addr := server.Listener.Addr().String()

	pool := newTestSparePool(t, server, clock.New(), 2, time.Minute)
	pool.touchAndTake(addr)
	pool.refill(addr)

	time.Sleep(300 * time.Millisecond)
	require.Equal(t, int32(2), handshakes.Load())
}
