package proxy

import (
	"context"
	"net/http"
	"net/url"
)

// Proxy defines the interface for request forwarding.
type Proxy interface {
	// Forward forwards a request to a backend target.
	Forward(ctx context.Context, target *url.URL, w http.ResponseWriter, r *http.Request) error
}

// LoadBalancer defines strategies for selecting backend targets.
type LoadBalancer interface {
	// SelectTarget returns the next target URL for load balancing.
	SelectTarget(ctx context.Context, targets []*url.URL) (*url.URL, error)
}
