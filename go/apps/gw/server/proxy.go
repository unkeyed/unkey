package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ProxyConfig configures the reverse proxy behavior.
type ProxyConfig struct {
	// Target is the backend URL to forward requests to
	Target *url.URL

	// Logger for debugging
	Logger logging.Logger

	// Timeout for backend requests
	Timeout time.Duration

	// ModifyRequest allows modifying the request before forwarding
	ModifyRequest func(*http.Request)

	// ModifyResponse allows modifying the response before returning to client
	ModifyResponse func(*http.Response) error
}

// NewReverseProxy creates a new reverse proxy handler with the given configuration.
func NewReverseProxy(config ProxyConfig) http.Handler {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Create a custom director function
	director := func(req *http.Request) {
		// Update the request to point to the target
		req.URL.Scheme = config.Target.Scheme
		req.URL.Host = config.Target.Host
		req.Host = config.Target.Host

		// Preserve the original path
		if config.Target.Path != "" {
			req.URL.Path = config.Target.Path + req.URL.Path
		}

		// Apply custom request modifications
		if config.ModifyRequest != nil {
			config.ModifyRequest(req)
		}

		// Log the forwarding
		if config.Logger != nil {
			config.Logger.Debug("forwarding request",
				"from", req.RemoteAddr,
				"to", req.URL.String(),
				"method", req.Method,
			)
		}
	}

	// Create the reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: config.Timeout,
		},
		ModifyResponse: config.ModifyResponse,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if config.Logger != nil {
				config.Logger.Error("proxy error",
					"error", err.Error(),
					"backend", config.Target.String(),
					"path", r.URL.Path,
				)
			}
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Bad Gateway"))
		},
	}

	return proxy
}

// SimpleProxy creates a basic reverse proxy that forwards all requests to a single backend.
func SimpleProxy(targetURL string, logger logging.Logger) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	return NewReverseProxy(ProxyConfig{
		Target: target,
		Logger: logger,
	}), nil
}

// RoundRobinProxy creates a reverse proxy that distributes requests across multiple backends.
type RoundRobinProxy struct {
	targets []*url.URL
	current int
	logger  logging.Logger
}

// NewRoundRobinProxy creates a new round-robin load balancing proxy.
func NewRoundRobinProxy(targets []string, logger logging.Logger) (*RoundRobinProxy, error) {
	urls := make([]*url.URL, len(targets))
	for i, target := range targets {
		u, err := url.Parse(target)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL %s: %w", target, err)
		}
		urls[i] = u
	}

	return &RoundRobinProxy{
		targets: urls,
		current: 0,
		logger:  logger,
	}, nil
}

// ServeHTTP implements http.Handler for round-robin load balancing.
func (rr *RoundRobinProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Select next target
	target := rr.targets[rr.current]
	rr.current = (rr.current + 1) % len(rr.targets)

	// Create proxy for this request
	proxy := NewReverseProxy(ProxyConfig{
		Target: target,
		Logger: rr.logger,
	})

	proxy.ServeHTTP(w, r)
}

// DebugHandler is a simple handler that echoes back request information.
// Useful for testing the gateway without a real backend.
func DebugHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "Gateway Debug Handler\n")
		fmt.Fprintf(w, "====================\n\n")
		fmt.Fprintf(w, "Method: %s\n", r.Method)
		fmt.Fprintf(w, "URL: %s\n", r.URL.String())
		fmt.Fprintf(w, "Host: %s\n", r.Host)
		fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
		fmt.Fprintf(w, "\nHeaders:\n")
		for k, v := range r.Header {
			fmt.Fprintf(w, "  %s: %v\n", k, v)
		}

		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				fmt.Fprintf(w, "\nBody:\n%s\n", string(body))
			}
		}
	}
}
