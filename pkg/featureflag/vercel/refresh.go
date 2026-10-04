package vercel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Diagnostics is a view of refresh health without flag, targeting, or key
// data.
type Diagnostics struct {
	Ready             bool
	SnapshotAge       time.Duration
	LastRefreshError  string
	LastRefreshFailed time.Time
	RefreshFailures   uint64
}

type health struct {
	mu       sync.Mutex
	lastErr  string
	failedAt time.Time
	failures atomic.Uint64
}

// Diagnostics returns the current refresh health.
func (p *Provider) Diagnostics() Diagnostics {
	p.health.mu.Lock()
	d := Diagnostics{
		Ready:             false,
		SnapshotAge:       0,
		LastRefreshError:  p.health.lastErr,
		LastRefreshFailed: p.health.failedAt,
		RefreshFailures:   p.health.failures.Load(),
	}
	p.health.mu.Unlock()

	if s := p.snapshot.Load(); s != nil {
		d.SnapshotAge = time.Since(s.fetchedAt)
		d.Ready = d.SnapshotAge <= p.config.MaxStaleness
	}
	return d
}

func (p *Provider) refresh(ctx context.Context) error {
	body, err := p.fetch(ctx)
	if err != nil {
		return p.recordFailure(err.Error())
	}
	s, err := parseDatafile(body, time.Now())
	if err != nil {
		return p.recordFailure("datafile parse")
	}

	p.snapshot.Store(s)
	p.health.mu.Lock()
	p.health.lastErr = ""
	p.health.failedAt = time.Time{}
	p.health.mu.Unlock()
	return nil
}

type refreshCause string

func (c refreshCause) Error() string { return string(c) }

func (p *Provider) fetch(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return nil, refreshCause("request")
	}
	req.Header.Set("Authorization", "Bearer "+p.config.SDKKey)

	res, err := p.client.Do(req)
	if err != nil {
		return nil, refreshCause("transport")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, refreshCause(fmt.Sprintf("http_%d", res.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes+1))
	if err != nil {
		return nil, refreshCause("body read")
	}
	if len(body) > maxBodyBytes {
		return nil, refreshCause("body too large")
	}
	return body, nil
}

func (p *Provider) recordFailure(cause string) error {
	p.health.failures.Add(1)
	p.health.mu.Lock()
	p.health.lastErr = cause
	p.health.failedAt = time.Now()
	p.health.mu.Unlock()
	return fmt.Errorf("vercel feature flags refresh failed: %s", cause)
}
