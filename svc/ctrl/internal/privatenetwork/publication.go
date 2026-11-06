package privatenetwork

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

type publication struct {
	platform string
	clock    clock.Clock

	mu             sync.Mutex
	snapshot       Snapshot
	published      bool
	certifiedUntil time.Time
	liveAt         time.Time
	ready          chan struct{}
}

func newPublication(platform string, clk clock.Clock) *publication {
	return &publication{
		platform:       platform,
		clock:          clk,
		mu:             sync.Mutex{},
		snapshot:       Snapshot{Connections: nil, Topology: nil, Version: "", ID: "", Certified: false},
		published:      false,
		certifiedUntil: time.Time{},
		liveAt:         time.Time{},
		ready:          make(chan struct{}),
	}
}

func (p *publication) markLive() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.liveAt = p.clock.Now()
}

func (p *publication) set(snapshot Snapshot, certifiedUntil time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.published {
		p.published = true
		close(p.ready)
	}
	p.snapshot = snapshot
	p.certifiedUntil = certifiedUntil
}

func (p *publication) get() (Snapshot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.clock.Now()
	if !p.published || now.Sub(p.liveAt) > liveWindow {
		return Snapshot{Connections: nil, Topology: nil, Version: "", ID: "", Certified: false}, ErrUnavailable
	}
	snapshot := p.snapshot
	remaining := p.certifiedUntil.Sub(now)
	snapshot.Certified = !p.certifiedUntil.IsZero() && remaining >= 0 && remaining <= certifiedAge
	certified := 0.0
	if snapshot.Certified {
		certified = 1
	}
	metrics.PrivateNetworkCatalogCertified.WithLabelValues(p.platform).Set(certified)
	return snapshot, nil
}

func (p *publication) wait(ctx context.Context) (Snapshot, error) {
	select {
	case <-p.ready:
		return p.get()
	case <-ctx.Done():
		return Snapshot{Connections: nil, Topology: nil, Version: "", ID: "", Certified: false}, ErrUnavailable
	}
}
