package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	upgradeSparesPerPeer = 4
	upgradeSpareIdle     = 5 * time.Minute
	upgradeSpareDialTime = 10 * time.Second
)

// sparePool keeps TLS connections to peer frontlines open before an upgrade
// needs them, so a cross-region upgrade skips the TCP and TLS round trips of
// a cold dial. A peer keeps spares only while it received an upgrade within
// idleAfter.
type sparePool struct {
	dialer    *tls.Dialer
	size      int
	idleAfter time.Duration

	mu    sync.Mutex
	peers map[string]*sparePeer
}

type sparePeer struct {
	spares   []*spareConn
	dialing  int
	lastUsed time.Time
	expiry   *time.Timer
}

func newSparePool(dialer *tls.Dialer, size int, idleAfter time.Duration) *sparePool {
	return &sparePool{
		dialer:    dialer,
		size:      size,
		idleAfter: idleAfter,
		mu:        sync.Mutex{},
		peers:     make(map[string]*sparePeer),
	}
}

// dial is an http.Transport DialTLSContext. It returns a live spare when one
// is ready and dials otherwise. Either way it tops the peer's spares back up.
func (p *sparePool) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	defer p.refill(addr)

	for spare := p.take(addr); spare != nil; spare = p.take(addr) {
		if spare.claim() {
			upgradeSparesTotal.WithLabelValues(spareOutcomeHit).Inc()
			return spare, nil
		}
	}

	upgradeSparesTotal.WithLabelValues(spareOutcomeMiss).Inc()
	return p.dialer.DialContext(ctx, network, addr)
}

// take marks addr as used and removes one of its spares, if any.
func (p *sparePool) take(addr string) *spareConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if !ok {
		//nolint:exhaustruct
		peer = &sparePeer{}
		peer.expiry = time.AfterFunc(p.idleAfter, func() { p.expire(addr) })
		p.peers[addr] = peer
	}
	peer.lastUsed = time.Now()

	n := len(peer.spares)
	if n == 0 {
		return nil
	}
	spare := peer.spares[n-1]
	peer.spares = peer.spares[:n-1]
	return spare
}

func (p *sparePool) refill(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if !ok {
		return
	}
	for ; len(peer.spares)+peer.dialing < p.size; peer.dialing++ {
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
	if !ok || !isTLS {
		_ = conn.Close()
		return
	}

	spare := newSpareConn(tlsConn)
	peer.spares = append(peer.spares, spare)
	go spare.watch(func() { p.drop(addr, spare) })
}

// drop removes a spare the peer closed and dials a replacement.
func (p *sparePool) drop(addr string, spare *spareConn) {
	_ = spare.Close()

	p.mu.Lock()
	peer, ok := p.peers[addr]
	if ok {
		peer.spares = slices.DeleteFunc(peer.spares, func(c *spareConn) bool { return c == spare })
	}
	p.mu.Unlock()

	if ok {
		p.refill(addr)
	}
}

// expire closes a peer's spares once it has gone idleAfter without an upgrade.
func (p *sparePool) expire(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if !ok {
		return
	}
	if idle := time.Since(peer.lastUsed); idle < p.idleAfter {
		peer.expiry.Reset(p.idleAfter - idle)
		return
	}
	delete(p.peers, addr)
	for _, spare := range peer.spares {
		_ = spare.Close()
	}
}
