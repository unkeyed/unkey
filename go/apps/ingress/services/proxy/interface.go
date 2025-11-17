package proxy

import (
	"context"
	"net/http"
	"time"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Service defines the interface for proxying requests to gateways or remote ingresses.
type Service interface {
	// ForwardToLocal forwards a request to a local gateway service (HTTP)
	ForwardToLocal(ctx context.Context, sess *zen.Session, deployment *partitionv1.Deployment, startTime time.Time) error

	// ForwardToRemote forwards a request to a remote ingress (HTTPS)
	ForwardToRemote(ctx context.Context, sess *zen.Session, targetRegion string, deployment *partitionv1.Deployment, startTime time.Time) error

	// GetMaxHops returns the maximum number of ingress hops allowed
	GetMaxHops() int
}

// Config holds configuration for the proxy service.
type Config struct {
	// Logger for debugging and monitoring
	Logger logging.Logger

	// IngressID is the current ingress instance ID
	IngressID string

	// Region is the current ingress region
	Region string

	// BaseDomain is the base domain for remote ingress routing (e.g., "aws.unkey.app")
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
