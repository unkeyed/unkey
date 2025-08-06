package proxy

import (
	"context"
	"net/http"
	"net/url"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Proxy defines the interface for request forwarding.
type Proxy interface {
	// Forward forwards a request to a backend target.
	Forward(ctx context.Context, target url.URL, w http.ResponseWriter, r *http.Request) error
}

// Config holds configuration for the proxy.
type Config struct {
	// Logger for debugging and monitoring
	Logger logging.Logger

	// MaxIdleConns is the maximum number of idle connections to keep open.
	MaxIdleConns int

	// IdleConnTimeout is the maximum amount of time an idle connection will remain open.
	IdleConnTimeout string

	// TLSHandshakeTimeout is the maximum amount of time a TLS handshake will take.
	TLSHandshakeTimeout string

	// Transport allows passing a shared HTTP transport for connection pooling
	// If nil, a new transport will be created with the other config values
	Transport *http.Transport
}
