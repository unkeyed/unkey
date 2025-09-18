package gateway_proxy

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/auth"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/apps/gw/services/validation"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
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

// Handle processes all HTTPS requests for the gateway.
// This is the main entry point for the gateway - all requests go through here.
func (h *Handler) Handle(ctx context.Context, sess *server.Session) error {
	ctx, span := tracing.Start(ctx, "proxy.handler")
	defer span.End()
	req := sess.Request()

	// Look up target configuration based on the request host
	// Strip port from hostname for database lookup (Host header may include port)
	hostname := routing.ExtractHostname(req)

	config, err := h.RoutingService.GetConfig(ctx, hostname)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Internal("failed to lookup target configuration"),
			fault.Public("Service configuration not found"),
		)
	}

	// Handle request validation if configured
	if h.Validator != nil {
		err = h.Validator.Validate(ctx, sess, config)
		if err != nil {
			return err
		}
	}

	// Handle authentication if configured
	err = h.Auth.Authenticate(ctx, sess, config)
	if err != nil {
		return err
	}

	// Select an available VM for this gateway
	targetURL, err := h.RoutingService.SelectVM(ctx, config)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Routing.VMSelectionFailed.URN()),
			fault.Internal("failed to select VM"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Forward the request using the proxy service with response capture
	captureWriter, captureFunc := sess.CaptureResponseWriter()
	err = h.Proxy.Forward(ctx, *targetURL, captureWriter, req)

	// Capture the response data back to session after forwarding
	captureFunc()

	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Proxy.ProxyForwardFailed.URN()),
			fault.Internal("failed to forward request"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	return nil
}
