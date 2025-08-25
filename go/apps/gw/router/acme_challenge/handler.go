package acme_challenge

import (
	"context"
	"log"
	"net"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/auth"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/apps/gw/services/validation"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Handler implements the main gateway proxy functionality.
// It handles all incoming requests and forwards them to backend services.
type Handler struct {
	Logger         logging.Logger
	RoutingService routing.Service
	Proxy          proxy.Proxy
	Auth           auth.Authenticator
	Validator      validation.Validator
}

// Handle sends all ACME Let's Encrypt challenges to the control plane.
func (h *Handler) Handle(ctx context.Context, sess *server.Session) error {
	req := sess.Request()

	// Log the incoming request
	h.Logger.Debug("handling gateway request",
		"requestId", sess.RequestID(),
		"method", req.Method,
		"host", req.Host,
		"path", req.URL.Path,
		"remoteAddr", req.RemoteAddr,
	)

	// Look up target configuration based on the request host
	// Strip port from hostname for database lookup (Host header may include port)
	hostname, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		// If SplitHostPort fails, req.Host doesn't contain a port, use it as-is
		hostname = req.Host
	}

	config, err := h.RoutingService.GetConfig(ctx, hostname)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Internal("failed to lookup target configuration"),
			fault.Public("Service configuration not found"),
		)
	}

	log.Printf("%#v", config)

	return nil
}
