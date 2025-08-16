package gateway_proxy

import (
	"context"
	"net"
	"time"

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

// Handle processes all HTTP requests for the gateway.
// This is the main entry point for the gateway - all requests go through here.
func (h *Handler) Handle(ctx context.Context, sess *server.Session) error {
	req := sess.Request()
	requestStart := time.Now()

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

	configLookupStart := time.Now()
	config, err := h.RoutingService.GetConfig(ctx, hostname)
	configLookupLatency := time.Since(configLookupStart)
	if err != nil {
		h.Logger.Error("failed to lookup target configuration",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"normalized_hostname", hostname,
			"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
			"config_lookup_latency_us", configLookupLatency.Microseconds(),
			"error", err.Error(),
		)

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
	vmSelectionStart := time.Now()
	targetURL, err := h.RoutingService.SelectVM(ctx, config)
	vmSelectionLatency := time.Since(vmSelectionStart)
	if err != nil {
		h.Logger.Error("failed to select VM",
			"requestId", sess.RequestID(),
			"deploymentID", config.DeploymentId,
			"host", req.Host,
			"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
			"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
			"error", err.Error(),
		)

		return fault.Wrap(err,
			fault.Code(codes.Gateway.Routing.VMSelectionFailed.URN()),
			fault.Internal("failed to select VM"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Calculate total routing overhead
	routingLatency := time.Since(requestStart)

	h.Logger.Debug("routing completed for request",
		"requestId", sess.RequestID(),
		"host", req.Host,
		"normalized_hostname", hostname,
		"deploymentID", config.DeploymentId,
		"selectedVM", targetURL.String(),
		"config_lookup_latency", configLookupLatency.String(),
		"vm_selection_latency", vmSelectionLatency.String(),
		"total_routing_latency", routingLatency.String(),
	)

	// Forward the request using the proxy service
	proxyStart := time.Now()
	err = h.Proxy.Forward(ctx, *targetURL, sess.ResponseWriter(), req)
	proxyLatency := time.Since(proxyStart)
	if err != nil {
		h.Logger.Error("failed to forward request",
			"requestId", sess.RequestID(),
			"deploymentID", config.DeploymentId,
			"selectedVM", targetURL.String(),
			"total_routing_latency", routingLatency.String(),
			"proxy_latency_ms", proxyLatency.String(),
			"error", err.Error(),
		)

		return fault.Wrap(err,
			fault.Code(codes.Gateway.Proxy.ProxyForwardFailed.URN()),
			fault.Internal("failed to forward request"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Log successful request completion with full timing breakdown
	totalLatency := time.Since(requestStart)
	h.Logger.Info("request completed successfully",
		"requestId", sess.RequestID(),
		"method", req.Method,
		"host", req.Host,
		"normalized_hostname", hostname,
		"path", req.URL.Path,
		"deploymentID", config.DeploymentId,
		"selectedVM", targetURL.String(),
		"config_lookup_latency_ms", configLookupLatency.String(),
		"vm_selection_latency_ms", vmSelectionLatency.String(),
		"total_routing_latency_ms", routingLatency.String(),
		"proxy_latency_ms", proxyLatency.String(),
		"total_request_latency_ms", totalLatency.String(),
		"routing_overhead_percent", float64(routingLatency.Microseconds())/float64(totalLatency.Microseconds())*100,
	)

	return nil
}
