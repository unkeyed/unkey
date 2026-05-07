package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

// Service defines the interface for proxying requests based on routing decisions.
type Service interface {
	// Forward dispatches a request based on the routing decision. Local
	// decisions go straight to the chosen instance; remote decisions hop
	// to the peer frontline in another region.
	Forward(ctx context.Context, sess *zen.Session, decision router.RouteDecision) error
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
