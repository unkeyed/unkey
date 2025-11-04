package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ Proxy = (*proxy)(nil)

type proxy struct {
	logger    logging.Logger
	transport *http.Transport
}

// New creates a new proxy with the given configuration.
func New(config Config) (Proxy, error) {
	// Use shared transport if provided, otherwise create a new one
	var transport *http.Transport
	if config.Transport != nil {
		transport = config.Transport
	} else {
		// Configure transport with defaults
		transport = &http.Transport{
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
	}

	return &proxy{
		logger:    config.Logger,
		transport: transport,
	}, nil
}

// classifyProxyError determines the appropriate error type based on the error content
func (p *proxy) classifyProxyError(err error) error {
	// Check for context cancellation (timeout)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Proxy.GatewayTimeout.URN()),
			fault.Internal("request timeout or cancellation"),
			fault.Public("The server took too long to respond"),
		)
	}

	// Check for net.Error timeout (handles most timeout cases)
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Proxy.GatewayTimeout.URN()),
			fault.Internal("upstream server timeout"),
			fault.Public("The server took too long to respond"),
		)
	}

	// Check for specific timeout strings
	errorStr := err.Error()
	if strings.Contains(errorStr, "timeout awaiting response headers") {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Proxy.GatewayTimeout.URN()),
			fault.Internal("upstream server timeout"),
			fault.Public("The server took too long to respond"),
		)
	}

	// Check for connection refused
	if strings.Contains(errorStr, "connection refused") {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Proxy.ServiceUnavailable.URN()),
			fault.Internal("backend service unavailable"),
			fault.Public("The service is currently unavailable"),
		)
	}

	// Default to bad gateway
	return fault.Wrap(err,
		fault.Code(codes.Gateway.Proxy.BadGateway.URN()),
		fault.Internal("proxy error occurred"),
		fault.Public("Unable to process the request"),
	)
}

// Forward implements the Proxy interface.
func (p *proxy) Forward(ctx context.Context, target url.URL, w http.ResponseWriter, r *http.Request) error {
	var err error

	// Create reverse proxy
	// nolint: exhaustruct
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
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, pErr error) {
			if p.logger != nil {
				p.logger.Error("proxy error",
					"error", pErr.Error(),
					"backend", target.String(),
					"path", r.URL.Path,
				)
			}

			// Classify the error and wrap it with appropriate fault
			err = p.classifyProxyError(pErr)
		},
	}

	// Execute the proxy
	proxy.ServeHTTP(w, r)

	return err
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
