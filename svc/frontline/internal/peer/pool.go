package peer

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"golang.org/x/net/http2"
)

// ErrNoConnection means no usable backhaul connection to the region exists.
var ErrNoConnection = errors.New("peer: no usable backhaul connection")

// PoolConfig configures a [Pool]. Zero values take the defaults.
type PoolConfig struct {
	// Regions are the peer regions to keep connections to, e.g. "us-east-1.aws".
	Regions   []string
	Discovery Discovery
	// Dial opens a TCP connection to a relay. Defaults to a net.Dialer with a
	// 5 s timeout and 15 s keepalive.
	Dial func(ctx context.Context, addr string) (net.Conn, error)

	// MinConns connections stay open per region, even when idle. A burst of new streams
	// on an idle connection is limited by TCP slow start (about 10 packets
	// per round trip), so burst capacity grows with the number of warm
	// connections, not with their size. Default 32.
	MinConns       int
	MaxConns       int           // default 64
	StreamsPerConn int           // load per connection that adds a connection; default 200
	PingInterval   time.Duration // default 5 s
	PingTimeout    time.Duration // default 2 s
	MaxAge         time.Duration // default 15 min, jittered ±20%
	Tick           time.Duration // maintenance interval; default 1 s
}

// Pool keeps warm, multiplexed HTTP/2 connections to relays in each peer
// region. Requests never wait for a dial: connections are opened at start,
// replaced before they expire, and added in the background under load.
type Pool struct {
	cfg       PoolConfig
	transport *http2.Transport

	mu      sync.Mutex
	regions map[string]*regionConns
	closed  bool

	kick chan struct{}
	stop chan struct{}
	wg   sync.WaitGroup
}

type regionConns struct {
	conns    []*poolConn
	dialing  map[string]int
	cooldown map[string]time.Time
	failures map[string]int
}

type poolConn struct {
	cc           *http2.ClientConn
	addr         string
	expires      time.Time
	draining     bool
	pingFailures int
}

// NewPool creates a pool. Call [Pool.Start] to open connections.
func NewPool(cfg PoolConfig) *Pool {
	if cfg.MinConns <= 0 {
		cfg.MinConns = 32
	}
	if cfg.MaxConns < cfg.MinConns {
		cfg.MaxConns = max(64, cfg.MinConns)
	}
	if cfg.StreamsPerConn <= 0 {
		cfg.StreamsPerConn = 200
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = 5 * time.Second
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = 2 * time.Second
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 15 * time.Minute
	}
	if cfg.Tick <= 0 {
		cfg.Tick = time.Second
	}
	if cfg.Dial == nil {
		//nolint:exhaustruct
		dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}
		cfg.Dial = func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", addr)
		}
	}

	regions := make(map[string]*regionConns, len(cfg.Regions))
	for _, region := range cfg.Regions {
		regions[region] = &regionConns{
			conns:    nil,
			dialing:  map[string]int{},
			cooldown: map[string]time.Time{},
			failures: map[string]int{},
		}
	}

	return &Pool{
		cfg: cfg,
		//nolint:exhaustruct
		transport: &http2.Transport{
			AllowHTTP:                  true,
			MaxReadFrameSize:           1 << 20,
			StrictMaxConcurrentStreams: true,
		},
		mu:      sync.Mutex{},
		regions: regions,
		closed:  false,
		kick:    make(chan struct{}, 1),
		stop:    make(chan struct{}),
		wg:      sync.WaitGroup{},
	}
}

// Start opens connections and runs maintenance and PINGs until Close.
func (p *Pool) Start() {
	p.wg.Add(2)
	go p.maintainLoop()
	go p.pingLoop()
}

// WaitReady blocks until every region has a usable connection or ctx ends.
func (p *Pool) WaitReady(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		ready := true
		for _, region := range p.cfg.Regions {
			if !p.Ready(region) {
				ready = false
				break
			}
		}
		if ready {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Close stops maintenance and closes every connection.
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.stop)
	var conns []*poolConn
	for _, rc := range p.regions {
		conns = append(conns, rc.conns...)
		rc.conns = nil
	}
	p.mu.Unlock()
	p.wg.Wait()
	for _, c := range conns {
		_ = c.cc.Close()
	}
}

// Ready reports whether region has a usable connection.
func (p *Pool) Ready(region string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	rc, ok := p.regions[region]
	if !ok {
		return false
	}
	return slices.ContainsFunc(rc.conns, usable)
}

// Get returns the less loaded of two random usable connections to region.
func (p *Pool) Get(region string) (*http2.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	rc, ok := p.regions[region]
	if !ok {
		return nil, ErrNoConnection
	}
	var live []*poolConn
	for _, c := range rc.conns {
		if usable(c) {
			live = append(live, c)
		}
	}
	if len(live) == 0 {
		p.kickLocked()
		return nil, ErrNoConnection
	}
	best := live[rand.IntN(len(live))]
	if len(live) > 1 {
		other := live[rand.IntN(len(live))]
		if load(other) < load(best) {
			best = other
		}
	}
	if load(best) >= p.cfg.StreamsPerConn/2 {
		p.kickLocked()
	}
	return best.cc, nil
}

func usable(c *poolConn) bool {
	return !c.draining && c.cc.CanTakeNewRequest()
}

func load(c *poolConn) int {
	s := c.cc.State()
	return s.StreamsActive + s.StreamsReserved + s.StreamsPending
}

func (p *Pool) kickLocked() {
	select {
	case p.kick <- struct{}{}:
	default:
	}
}

func (p *Pool) maintainLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.cfg.Tick)
	defer ticker.Stop()
	for {
		p.maintain()
		select {
		case <-p.stop:
			return
		case <-ticker.C:
		case <-p.kick:
		}
	}
}

func (p *Pool) maintain() {
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	for region, rc := range p.regions {
		kept := rc.conns[:0]
		for _, c := range rc.conns {
			state := c.cc.State()
			if state.Closed {
				connectionEventsTotal.WithLabelValues(region, eventClosed).Inc()
				continue
			}
			if !c.draining && (state.Closing || now.After(c.expires)) {
				c.draining = true
				if !state.Closing {
					connectionEventsTotal.WithLabelValues(region, eventMaxAge).Inc()
				}
				go shutdown(c.cc)
			}
			kept = append(kept, c)
		}
		rc.conns = kept

		live, streams := 0, 0
		for _, c := range rc.conns {
			if usable(c) {
				live++
				streams += load(c)
			}
		}
		connectionsGauge.WithLabelValues(region).Set(float64(live))

		perConn := max(1, p.cfg.StreamsPerConn/2)
		want := min(max(p.cfg.MinConns, (streams+perConn-1)/perConn+1), p.cfg.MaxConns)
		dialing := 0
		for _, n := range rc.dialing {
			dialing += n
		}
		for i := live + dialing; i < want; i++ {
			addr, ok := p.pickAddrLocked(region, rc, now)
			if !ok {
				break
			}
			rc.dialing[addr]++
			go p.dial(region, addr)
		}
	}
}

// pickAddrLocked prefers the relay with the fewest connections from this
// pod, so connections spread over relays and, with WireGuard, over node pairs.
func (p *Pool) pickAddrLocked(region string, rc *regionConns, now time.Time) (string, bool) {
	addrs := p.cfg.Discovery.Addresses(region)
	best, bestCount := "", -1
	for _, i := range rand.Perm(len(addrs)) {
		addr := addrs[i]
		if now.Before(rc.cooldown[addr]) {
			continue
		}
		count := rc.dialing[addr]
		for _, c := range rc.conns {
			if c.addr == addr && !c.draining {
				count++
			}
		}
		if bestCount == -1 || count < bestCount {
			best, bestCount = addr, count
		}
	}
	return best, bestCount != -1
}

func (p *Pool) dial(region, addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cc, err := p.open(ctx, addr)

	p.mu.Lock()
	defer p.mu.Unlock()
	rc := p.regions[region]
	rc.dialing[addr]--
	if rc.dialing[addr] <= 0 {
		delete(rc.dialing, addr)
	}
	if err != nil {
		connectionEventsTotal.WithLabelValues(region, eventDialError).Inc()
		rc.failures[addr]++
		backoff := min(100*time.Millisecond<<min(rc.failures[addr], 6), 5*time.Second)
		rc.cooldown[addr] = time.Now().Add(backoff/2 + rand.N(backoff/2+1))
		logger.Warn("backhaul dial failed", "region", region, "addr", addr, "error", err.Error())
		return
	}
	if p.closed {
		_ = cc.Close()
		return
	}
	connectionEventsTotal.WithLabelValues(region, eventDialOK).Inc()
	delete(rc.failures, addr)
	delete(rc.cooldown, addr)
	jitter := time.Duration(rand.Int64N(int64(p.cfg.MaxAge)*2/5+1)) - p.cfg.MaxAge/5
	rc.conns = append(rc.conns, &poolConn{
		cc:           cc,
		addr:         addr,
		expires:      time.Now().Add(p.cfg.MaxAge + jitter),
		draining:     false,
		pingFailures: 0,
	})
}

func (p *Pool) open(ctx context.Context, addr string) (*http2.ClientConn, error) {
	conn, err := p.cfg.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	cc, err := p.transport.NewClientConn(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := cc.Ping(ctx); err != nil {
		_ = cc.Close()
		return nil, err
	}
	return cc, nil
}

// shutdown sends no new streams on cc and closes it once its streams end.
// WebSockets can keep a draining connection open for as long as they live.
func shutdown(cc *http2.ClientConn) {
	_ = cc.Shutdown(context.Background())
}

func (p *Pool) pingLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
		}
		type target struct {
			region string
			conn   *poolConn
		}
		var targets []target
		p.mu.Lock()
		for region, rc := range p.regions {
			for _, c := range rc.conns {
				if !c.draining {
					targets = append(targets, target{region: region, conn: c})
				}
			}
		}
		p.mu.Unlock()

		var wg sync.WaitGroup
		for _, t := range targets {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.ping(t.region, t.conn)
			}()
		}
		wg.Wait()
	}
}

func (p *Pool) ping(region string, c *poolConn) {
	ctx, cancel := context.WithTimeout(context.Background(), p.cfg.PingTimeout)
	defer cancel()
	start := time.Now()
	err := c.cc.Ping(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		c.pingFailures = 0
		pingRTTSeconds.WithLabelValues(region).Observe(time.Since(start).Seconds())
		return
	}
	c.pingFailures++
	if c.pingFailures >= 2 {
		connectionEventsTotal.WithLabelValues(region, eventPingTimeout).Inc()
		_ = c.cc.Close()
		p.kickLocked()
	}
}
