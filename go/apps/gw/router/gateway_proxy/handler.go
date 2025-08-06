package gateway_proxy

import (
	"context"
	"net"
	"time"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
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
	targetInfo, err := h.RoutingService.GetConfig(ctx, hostname)
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
		return err
	}

	// Associate the workspace ID with the session for metrics/logging
	// EHH TODO:
	// sess.WorkspaceID = targetInfo.WorkspaceID

	// Select an available VM for this gateway
	vmSelectionStart := time.Now()
	targetURL, err := h.RoutingService.SelectVM(ctx, targetInfo)
	vmSelectionLatency := time.Since(vmSelectionStart)
	if err != nil {
		h.Logger.Error("failed to select VM",
			"requestId", sess.RequestID(),
			"deploymentID", targetInfo.DeploymentId,
			"host", req.Host,
			"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
			"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
			"error", err.Error(),
		)
		return err
	}

	// Calculate total routing overhead
	routingLatency := time.Since(requestStart)

	h.Logger.Debug("routing completed for request",
		"requestId", sess.RequestID(),
		"host", req.Host,
		"normalized_hostname", hostname,
		"deploymentID", targetInfo.DeploymentId,
		"selectedVM", targetURL.String(),
		"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
		"config_lookup_latency_us", configLookupLatency.Microseconds(),
		"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
		"vm_selection_latency_us", vmSelectionLatency.Microseconds(),
		"total_routing_latency_ms", routingLatency.Milliseconds(),
		"total_routing_latency_us", routingLatency.Microseconds(),
	)

	// Forward the request using the proxy service
	proxyStart := time.Now()
	err = h.Proxy.Forward(ctx, targetURL, sess.ResponseWriter(), req)
	proxyLatency := time.Since(proxyStart)
	if err != nil {
		h.Logger.Error("failed to forward request",
			"requestId", sess.RequestID(),
			"deploymentID", targetInfo.DeploymentId,
			"selectedVM", targetURL.String(),
			"total_routing_latency_ms", routingLatency.Milliseconds(),
			"proxy_latency_ms", proxyLatency.Milliseconds(),
			"error", err.Error(),
		)

		return fault.Wrap(err,
			fault.Code(codes.Gateway.BadRequest.BadGateway.URN()),
			fault.Internal("something went wrong proxying the request"),
			fault.Public("We're unable to forward the request"),
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
		"deploymentID", targetInfo.DeploymentId,
		"selectedVM", targetURL.String(),
		"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
		"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
		"total_routing_latency_ms", routingLatency.Milliseconds(),
		"proxy_latency_ms", proxyLatency.Milliseconds(),
		"total_request_latency_ms", totalLatency.Milliseconds(),
		"routing_overhead_percent", float64(routingLatency.Microseconds())/float64(totalLatency.Microseconds())*100,
	)

	return nil
}
