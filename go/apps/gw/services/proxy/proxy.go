package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ Proxy = (*proxy)(nil)

// proxy implements the Proxy interface with support for single or multiple backend URLs.
type proxy struct {
	targets   []target
	logger    logging.Logger
	balancer  LoadBalancer
	transport *http.Transport
}

// target represents a backend target with its URL and optional configuration.
type target struct {
	url    *url.URL
	weight int // For weighted load balancing (future use)
}

// New creates a new proxy with the given configuration.
func New(config Config) (Proxy, error) {
	// Allow empty targets for dynamic routing scenarios
	// Targets will be provided per-request via the Forward method

	targets := make([]target, len(config.Targets))
	for i, targetURL := range config.Targets {
		u, err := url.Parse(targetURL)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL %s: %w", targetURL, err)
		}
		targets[i] = target{
			url:    u,
			weight: 1, // Default weight
		}
	}

	// Default load balancer is round-robin
	balancer := config.LoadBalancer
	if balancer == nil {
		balancer = NewRoundRobinBalancer()
	}

	// Configure transport with defaults
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	// Apply config overrides if provided
	if config.MaxIdleConns > 0 {
		transport.MaxIdleConns = config.MaxIdleConns
	}

	if config.IdleConnTimeout != "" {
		if timeout, err := time.ParseDuration(config.IdleConnTimeout); err == nil {
			transport.IdleConnTimeout = timeout
		}
	}

	if config.TLSHandshakeTimeout != "" {
		if timeout, err := time.ParseDuration(config.TLSHandshakeTimeout); err == nil {
			transport.TLSHandshakeTimeout = timeout
		}
	}

	return &proxy{
		targets:   targets,
		logger:    config.Logger,
		balancer:  balancer,
		transport: transport,
	}, nil
}

// Forward implements the Proxy interface.
func (p *proxy) Forward(ctx context.Context, target *url.URL, w http.ResponseWriter, r *http.Request) error {
	// If no specific target is provided, use load balancer to select one
	if target == nil {
		if len(p.targets) == 0 {
			return fmt.Errorf("no target URL provided and no default targets configured")
		}

		targetURLs := make([]*url.URL, len(p.targets))
		for i, t := range p.targets {
			targetURLs[i] = t.url
		}

		selectedTarget, err := p.balancer.SelectTarget(ctx, targetURLs)
		if err != nil {
			return fmt.Errorf("failed to select target: %w", err)
		}
		target = selectedTarget
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Update the request to point to the target
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Preserve the path but prepend target path if it exists
			if target.Path != "" && target.Path != "/" {
				req.URL.Path = strings.TrimSuffix(target.Path, "/") + req.URL.Path
			}

			// Add forwarding headers
			if clientIP := getClientIP(req); clientIP != "" {
				req.Header.Set("X-Forwarded-For", clientIP)
			}

			req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
			req.Header.Set("X-Forwarded-Host", req.Host)

			if p.logger != nil {
				p.logger.Debug("forwarding request",
					"from", req.RemoteAddr,
					"to", req.URL.String(),
					"method", req.Method,
					"path", req.URL.Path,
				)
			}
		},
		Transport: p.transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if p.logger != nil {
				p.logger.Error("proxy error",
					"error", err.Error(),
					"backend", target.String(),
					"path", r.URL.Path,
				)
			}

			// we handle errors ourselves

			// w.WriteHeader(http.StatusBadGateway)
			// w.Write([]byte("Bad Gateway"))
		},
	}

	// Execute the proxy
	proxy.ServeHTTP(w, r)

	return nil
}

// getClientIP extracts the client IP from various headers.
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if there are multiple
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}

		return strings.TrimSpace(xff)
	}

	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
