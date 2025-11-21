package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Service defines the interface for proxying requests to gateways or remote NLBs.
type Service interface {
	// ForwardToGateway forwards a request to a local gateway service (HTTP)
	// Adds X-Unkey-Deployment-Id header for the gateway to route to the correct deployment
	// Request start time is retrieved from context
	ForwardToGateway(ctx context.Context, sess *zen.Session, gateway *db.Gateway, deploymentID string) error

	// ForwardToNLB forwards a request to a remote region's NLB (HTTPS)
	// Keeps the original hostname so the remote ingress can do TLS termination and routing
	// Request start time is retrieved from context
	ForwardToNLB(ctx context.Context, sess *zen.Session, targetRegion string) error
}

// Config holds configuration for the proxy service.
type Config struct {
	// Logger for debugging and monitoring
	Logger logging.Logger

	// IngressID is the current ingress instance ID
	IngressID string

	// Region is the current ingress region
	Region string

	// BaseDomain is the base domain for remote NLB routing (e.g., "unkey.cloud")
	BaseDomain string

	// Clock for time tracking
	Clock clock.Clock

	// MaxHops is the maximum number of ingress hops allowed before rejecting the request.
	// If 0, defaults to 3.
	MaxHops int

	// MaxIdleConns is the maximum number of idle connections to keep open.
	MaxIdleConns int

	// IdleConnTimeout is the maximum amount of time an idle connection will remain open.
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout is the maximum amount of time a TLS handshake will take.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout is the maximum amount of time to wait for response headers.
	ResponseHeaderTimeout time.Duration

	// Transport allows passing a shared HTTP transport for connection pooling
	// If nil, a new transport will be created with the other config values
	Transport *http.Transport
}
