package gateway_proxy

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Handler implements the main gateway proxy functionality.
// It handles all incoming requests and forwards them to backend services.
type Handler struct {
	Logger         logging.Logger
	RoutingService routing.Service
}

// Handle processes all HTTP requests for the gateway.
// This is the main entry point for the gateway - all requests go through here.
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
	targetInfo, err := h.RoutingService.GetTargetByHost(ctx, req.Host)
	if err != nil {
		h.Logger.Error("failed to lookup target configuration",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"error", err.Error(),
		)
		return err
	}

	// Associate the workspace ID with the session for metrics/logging
	// sess.WorkspaceID = targetInfo.WorkspaceID

	// Select an available VM for this gateway
	targetURL, err := h.RoutingService.SelectVM(ctx, targetInfo)
	if err != nil {
		h.Logger.Error("failed to select VM",
			"requestId", sess.RequestID(),
			"gatewayID", targetInfo.GatewayID,
			"host", req.Host,
			"error", err.Error(),
		)
		return err
	}

	h.Logger.Debug("selected VM for request",
		"requestId", sess.RequestID(),
		"gatewayID", targetInfo.GatewayID,
		"selectedVM", targetURL.String(),
	)

	// Create a proxy for this specific target
	proxyService, err := proxy.SingleTargetProxy(targetURL.String(), h.Logger)
	if err != nil {
		h.Logger.Error("failed to create proxy",
			"requestId", sess.RequestID(),
			"target", targetURL.String(),
			"error", err.Error(),
		)
		return err
	}

	// Forward the request to the resolved target
	err = proxyService.Forward(ctx, targetURL, sess.ResponseWriter(), req)
	if err != nil {
		h.Logger.Error("failed to forward request",
			"requestId", sess.RequestID(),
			"gatewayID", targetInfo.GatewayID,
			"selectedVM", targetURL.String(),
			"error", err.Error(),
		)
		return err
	}

	// No need to write a response here - the proxy has already handled it
	return nil
}
