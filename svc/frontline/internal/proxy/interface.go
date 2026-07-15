package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
)

// Service forwards HTTP requests to either a local deployment instance or
// a peer frontline in another region. Callers pick the target; the service
// itself is unaware of routing policy.
type Service interface {
	// ForwardToInstance proxies the request to the given deployment instance
	// using the transport selected by protocol. Returns the wrapped proxy
	// error on failure; callers may use [IsDialError] to decide whether the
	// request is safe to replay against a different instance.
	ForwardToInstance(ctx context.Context, sess *zen.Session, protocol db.DeploymentsUpstreamProtocol, instance db.FindInstancesByDeploymentIDRow) error

	// ForwardToRegion proxies the request to the peer frontline serving
	// targetRegionPlatform (e.g. "us-east-1.aws"). The peer redoes the full
	// hostname → engine → instance chain and is responsible for its own
	// retry and logging.
	ForwardToRegion(ctx context.Context, sess *zen.Session, targetRegionPlatform string) error
}

// Config holds configuration for the proxy service.
type Config struct {
	// InstanceID is the current frontline instance ID
	InstanceID string

	Platform string
	// Region is the current frontline region
	Region string

	// ApexDomain is the apex domain for remote NLB routing (e.g., "unkey.cloud")
	// Routes to frontline.{region}.{ApexDomain} (e.g., frontline.us-east-1.aws.unkey.cloud)
	ApexDomain string

	// Clock for time tracking
	Clock clock.Clock

	// MaxHops is the maximum number of frontline hops allowed before rejecting the request.
	// If 0, defaults to 3.
	MaxHops int

	// MaxIdleConns is the maximum number of idle connections to keep open.
	MaxIdleConns int

	// IdleConnTimeout is the maximum amount of time an idle connection will remain open.
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout is the maximum amount of time a TLS handshake will take.
	TLSHandshakeTimeout time.Duration

	// Transport allows passing a shared HTTP transport for connection pooling
	// for the cross-region peer hop. If nil, a new transport will be created.
	Transport *http.Transport

	// UpstreamTransports holds transports for each supported upstream protocol
	// (http1, h2c) used when forwarding to a deployment instance. Required.
	UpstreamTransports *TransportRegistry

	// ErrorPageRenderer renders HTML error pages.
	ErrorPageRenderer errorpage.Renderer
}
