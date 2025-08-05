package proxy

import (
	"context"
	"fmt"
	"net/url"
	"sync"
)

// RoundRobinBalancer implements a simple round-robin load balancing strategy.
type RoundRobinBalancer struct {
	mu      sync.Mutex
	current int
}

// NewRoundRobinBalancer creates a new round-robin load balancer.
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

// SelectTarget implements the LoadBalancer interface.
func (rr *RoundRobinBalancer) SelectTarget(ctx context.Context, targets []*url.URL) (*url.URL, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets available")
	}

	rr.mu.Lock()
	defer rr.mu.Unlock()

	target := targets[rr.current]
	rr.current = (rr.current + 1) % len(targets)

	return target, nil
}
