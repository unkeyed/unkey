package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"golang.org/x/net/http/httpguts"
)

const (
	upgradeSparesPerPeer = 4
	upgradeSpareIdle     = 5 * time.Minute
	upgradeSpareDialTime = 10 * time.Second
)

// sparePool keeps TLS connections to peer frontlines open before a request
// needs them. A protocol upgrade takes over its connection for the whole
// session, so it cannot use the pooled HTTP/2 connection, and a cold dial
// costs a TCP and a TLS round trip across regions before the upgrade starts.
//
// Spares exist only for peers that received an upgrade within idleAfter.
type sparePool struct {
	dialer    *tls.Dialer
	size      int
	idleAfter time.Duration

	mu     sync.Mutex
	peers  map[string]*sparePeer
	closed bool
}

type sparePeer struct {
	conns    []*spareConn
	dialing  int
	lastUsed time.Time
	expiry   *time.Timer
}

type spareConn struct {
	conn    *tls.Conn
	watched chan struct{}
	dead    atomic.Bool
}

func newSparePool(dialer *tls.Dialer, size int, idleAfter time.Duration) *sparePool {
	return &sparePool{
		dialer:    dialer,
		size:      size,
		idleAfter: idleAfter,
		mu:        sync.Mutex{},
		peers:     make(map[string]*sparePeer),
		closed:    false,
	}
}

// upgradeTransport sends HTTP/1.1 requests over spare connections. The peer
// can close a spare while its FIN is still in flight, and http.Transport
// retries only on reused pooled connections, so a request without a body that
// failed on a spare is retried once on a fresh dial.
type upgradeTransport struct {
	spares *http.Transport
	fresh  *http.Transport
}

// spareTLSConn marks a connection that came from the spare pool.
type spareTLSConn struct {
	*tls.Conn
}

func newUpgradeTransport(spares *sparePool) *upgradeTransport {
	h1 := func(dial func(context.Context, string, string) (net.Conn, error)) *http.Transport {
		//nolint:exhaustruct
		return &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialTLSContext:        dial,
			TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
			MaxIdleConnsPerHost:   upgradeSparesPerPeer,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	return &upgradeTransport{
		spares: h1(spares.DialTLSContext),
		fresh:  h1(spares.dialer.DialContext),
	}
}

func (t *upgradeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	usedSpare := false
	//nolint:exhaustruct
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			_, usedSpare = info.Conn.(*spareTLSConn)
		},
	}
	resp, err := t.spares.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
	if err == nil || !usedSpare || req.ContentLength != 0 || req.Context().Err() != nil {
		return resp, err
	}
	upgradeSparesTotal.WithLabelValues(spareOutcomeStale).Inc()
	return t.fresh.RoundTrip(req)
}

func newUpgradeDialer() *tls.Dialer {
	//nolint:exhaustruct
	return &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout:   upgradeSpareDialTime,
			KeepAlive: 30 * time.Second,
		},
		Config: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
		},
	}
}

func isUpgradeRequest(req *http.Request) bool {
	return httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") &&
		req.Header.Get("Upgrade") != ""
}

func (p *sparePool) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	defer p.refill(addr)

	for {
		spare := p.take(addr)
		if spare == nil {
			break
		}
		if spare.claim() {
			upgradeSparesTotal.WithLabelValues(spareOutcomeHit).Inc()
			return &spareTLSConn{Conn: spare.conn}, nil
		}
	}

	upgradeSparesTotal.WithLabelValues(spareOutcomeMiss).Inc()
	return p.dialer.DialContext(ctx, network, addr)
}

func (p *sparePool) take(addr string) *spareConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	peer, ok := p.peers[addr]
	if !ok {
		//nolint:exhaustruct
		peer = &sparePeer{}
		peer.expiry = time.AfterFunc(p.idleAfter, func() { p.expire(addr) })
		p.peers[addr] = peer
	}
	peer.lastUsed = time.Now()

	if len(peer.conns) == 0 {
		return nil
	}
	spare := peer.conns[len(peer.conns)-1]
	peer.conns = peer.conns[:len(peer.conns)-1]
	return spare
}

func (p *sparePool) refill(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if !ok || p.closed {
		return
	}
	for ; len(peer.conns)+peer.dialing < p.size; peer.dialing++ {
		go p.dialSpare(addr)
	}
}

func (p *sparePool) dialSpare(addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), upgradeSpareDialTime)
	conn, err := p.dialer.DialContext(ctx, "tcp", addr)
	cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if ok {
		peer.dialing--
	}
	if err != nil {
		logger.Debug("unable to dial spare peer connection", "addr", addr, "error", err.Error())
		return
	}
	tlsConn, isTLS := conn.(*tls.Conn)
	if !isTLS || !ok || p.closed {
		_ = conn.Close()
		return
	}

	spare := &spareConn{conn: tlsConn, watched: make(chan struct{}), dead: atomic.Bool{}}
	peer.conns = append(peer.conns, spare)
	go spare.watch(func() { p.drop(addr, spare) })
}

func (p *sparePool) drop(addr string, spare *spareConn) {
	_ = spare.conn.Close()

	p.mu.Lock()
	peer, ok := p.peers[addr]
	if ok {
		peer.conns = slices.DeleteFunc(peer.conns, func(c *spareConn) bool { return c == spare })
	}
	p.mu.Unlock()

	if ok {
		p.refill(addr)
	}
}

func (p *sparePool) expire(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if !ok {
		return
	}
	idle := time.Since(peer.lastUsed)
	if idle < p.idleAfter {
		peer.expiry.Reset(p.idleAfter - idle)
		return
	}
	delete(p.peers, addr)
	for _, spare := range peer.conns {
		_ = spare.conn.Close()
	}
}

func (p *sparePool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	for addr, peer := range p.peers {
		peer.expiry.Stop()
		for _, spare := range peer.conns {
			_ = spare.conn.Close()
		}
		delete(p.peers, addr)
	}
}

// watch blocks in Read until the peer closes the connection or claim
// interrupts it. Reading through the TLS layer consumes post-handshake
// messages such as session tickets without losing record state.
func (c *spareConn) watch(onDead func()) {
	defer close(c.watched)

	var b [1]byte
	_, err := c.conn.Read(b[:])
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return
	}
	c.dead.Store(true)
	onDead()
}

// claim stops the watcher and reports whether the connection is still usable.
// A TLS read timeout is not sticky, so the connection stays usable after it.
func (c *spareConn) claim() bool {
	_ = c.conn.SetReadDeadline(time.Unix(1, 0))
	<-c.watched
	if c.dead.Load() {
		return false
	}
	_ = c.conn.SetReadDeadline(time.Time{})
	return true
}
