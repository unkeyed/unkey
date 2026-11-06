package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	upgradeSparesPerPeer = 4
	upgradeSpareIdle     = 5 * time.Minute
	// A spare that the peer closes sooner than this is replaced only by the
	// next upgrade, so a peer that closes every connection right after the
	// handshake cannot cause a redial loop.
	upgradeSpareMinLife = 5 * time.Second
)

// sparePool keeps TLS connections to peer frontlines open before an upgrade
// needs them, so a cross-region upgrade skips the TCP and TLS round trips of
// a cold dial. A peer keeps spares only while it received an upgrade within
// idleAfter.
type sparePool struct {
	dialer    *tls.Dialer
	clock     clock.Clock
	size      int
	idleAfter time.Duration

	mu    sync.Mutex
	peers map[string]*sparePeer
}

type sparePeer struct {
	spares   []*spareConn
	dialing  int
	lastUsed time.Time
}

func newSparePool(dialer *tls.Dialer, clk clock.Clock, size int, idleAfter time.Duration) *sparePool {
	p := &sparePool{
		dialer:    dialer,
		clock:     clk,
		size:      size,
		idleAfter: idleAfter,
		mu:        sync.Mutex{},
		peers:     make(map[string]*sparePeer),
	}
	go p.expireIdle(clk.NewTicker(idleAfter / 2))
	return p
}

// dial is an http.Transport DialTLSContext. It returns a live spare when one
// is ready and dials otherwise. Either way it tops the peer's spares back up.
func (p *sparePool) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	defer p.refill(addr)

	for spare := p.touchAndTake(addr); spare != nil; spare = p.touchAndTake(addr) {
		if spare.claim() {
			return spare, nil
		}
	}
	return p.dialer.DialContext(ctx, network, addr)
}

// touchAndTake marks addr as used, starting to track it if needed, and
// removes one of its spares, if any.
func (p *sparePool) touchAndTake(addr string) *spareConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	peer, ok := p.peers[addr]
	if !ok {
		//nolint:exhaustruct
		peer = &sparePeer{}
		p.peers[addr] = peer
	}
	peer.lastUsed = p.clock.Now()

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
		go p.dialSpare(addr, peer)
	}
}

func (p *sparePool) dialSpare(addr string, peer *sparePeer) {
	conn, err := p.dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		upstreamDialsTotal.WithLabelValues(destinationFrontline, dialOutcomeError).Inc()
	} else {
		upstreamDialsTotal.WithLabelValues(destinationFrontline, dialOutcomeSuccess).Inc()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	peer.dialing--
	if err != nil {
		logger.Debug("unable to dial spare peer connection", "addr", addr, "error", err.Error())
		return
	}
	tlsConn, isTLS := conn.(*tls.Conn)
	if !isTLS || p.peers[addr] != peer {
		_ = conn.Close()
		return
	}

	spare := newSpareConn(tlsConn, p.clock.Now())
	peer.spares = append(peer.spares, spare)
	go spare.watch(func() { p.drop(addr, peer, spare) })
}

// drop removes a spare the peer closed and dials a replacement.
func (p *sparePool) drop(addr string, peer *sparePeer, spare *spareConn) {
	_ = spare.Close()

	p.mu.Lock()
	peer.spares = slices.DeleteFunc(peer.spares, func(c *spareConn) bool { return c == spare })
	current := p.peers[addr] == peer
	p.mu.Unlock()

	if current && p.clock.Now().Sub(spare.born) >= upgradeSpareMinLife {
		p.refill(addr)
	}
}

// expireIdle closes the spares of peers that went idleAfter without an upgrade.
func (p *sparePool) expireIdle(ticker clock.Ticker) {
	for range ticker.C() {
		now := p.clock.Now()
		p.mu.Lock()
		for addr, peer := range p.peers {
			if now.Sub(peer.lastUsed) < p.idleAfter {
				continue
			}
			delete(p.peers, addr)
			for _, spare := range peer.spares {
				_ = spare.Close()
			}
		}
		p.mu.Unlock()
	}
}
