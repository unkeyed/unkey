package gateway_proxy

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Handler implements the main gateway proxy functionality.
// It handles all incoming requests and forwards them to backend services.
type Handler struct {
	Logger         logging.Logger
	RoutingService routing.Service
	Proxy          proxy.Proxy
	Keys           keys.KeyService
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

	// Verify API key from Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		h.Logger.Warn("missing authorization header",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"path", req.URL.Path,
		)
		http.Error(sess.ResponseWriter(), "Authorization header required", http.StatusUnauthorized)
		return nil
	}

	// Extract Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		h.Logger.Warn("invalid authorization header format",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"path", req.URL.Path,
		)
		http.Error(sess.ResponseWriter(), "Invalid authorization header format", http.StatusUnauthorized)
		return nil
	}

	rootKey := strings.TrimPrefix(authHeader, "Bearer ")
	if rootKey == "" {
		h.Logger.Warn("empty api key",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"path", req.URL.Path,
		)
		http.Error(sess.ResponseWriter(), "API key is required", http.StatusUnauthorized)
		return nil
	}

	// The keys service needs this for workspace tracking
	zenSess := &zen.Session{}

	// Verify the API key
	keyVerifyStart := time.Now()
	key, logKeyFunc, err := h.Keys.Get(ctx, zenSess, rootKey)
	defer logKeyFunc()
	keyVerifyLatency := time.Since(keyVerifyStart)

	if err != nil {
		h.Logger.Error("failed to verify api key",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"path", req.URL.Path,
			"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
			"error", err.Error(),
		)
		http.Error(sess.ResponseWriter(), "Internal server error", http.StatusInternalServerError)
		return nil
	}

	// Check if key is valid
	if key.Status != keys.StatusValid {
		h.Logger.Warn("invalid api key",
			"requestId", sess.RequestID(),
			"host", req.Host,
			"path", req.URL.Path,
			"key_status", key.Status,
			"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
		)
		http.Error(sess.ResponseWriter(), "Invalid API key", http.StatusUnauthorized)
		return nil
	}

	// API key is valid, continue with request
	h.Logger.Debug("api key verified successfully",
		"requestId", sess.RequestID(),
		"host", req.Host,
		"path", req.URL.Path,
		"key_id", key.Key.ID,
		"workspace_id", key.Key.WorkspaceID,
		"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
	)

	// Associate the workspace ID with the session for metrics/logging
	sess.WorkspaceID = key.Key.WorkspaceID

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
			"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
			"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
			"error", err.Error(),
		)
		return err
	}

	// Calculate total routing overhead (including key verification)
	routingLatency := time.Since(requestStart)

	h.Logger.Debug("routing completed for request",
		"requestId", sess.RequestID(),
		"host", req.Host,
		"normalized_hostname", hostname,
		"deploymentID", targetInfo.DeploymentId,
		"selectedVM", targetURL.String(),
		"key_id", key.Key.ID,
		"workspace_id", key.Key.WorkspaceID,
		"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
		"config_lookup_latency_us", configLookupLatency.Microseconds(),
		"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
		"key_verify_latency_us", keyVerifyLatency.Microseconds(),
		"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
		"vm_selection_latency_us", vmSelectionLatency.Microseconds(),
		"total_routing_latency_ms", routingLatency.Milliseconds(),
		"total_routing_latency_us", routingLatency.Microseconds(),
	)

	// Forward the request using the proxy service
	proxyStart := time.Now()
	err = h.Proxy.Forward(ctx, *targetURL, sess.ResponseWriter(), req)
	proxyLatency := time.Since(proxyStart)
	if err != nil {
		h.Logger.Error("failed to forward request",
			"requestId", sess.RequestID(),
			"deploymentID", targetInfo.DeploymentId,
			"selectedVM", targetURL.String(),
			"key_id", key.Key.ID,
			"workspace_id", key.Key.WorkspaceID,
			"total_routing_latency_ms", routingLatency.Milliseconds(),
			"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
			"proxy_latency_ms", proxyLatency.Milliseconds(),
			"error", err.Error(),
		)

		return err
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
		"key_id", key.Key.ID,
		"workspace_id", key.Key.WorkspaceID,
		"config_lookup_latency_ms", configLookupLatency.Milliseconds(),
		"key_verify_latency_ms", keyVerifyLatency.Milliseconds(),
		"vm_selection_latency_ms", vmSelectionLatency.Milliseconds(),
		"total_routing_latency_ms", routingLatency.Milliseconds(),
		"proxy_latency_ms", proxyLatency.Milliseconds(),
		"total_request_latency_ms", totalLatency.Milliseconds(),
		"routing_overhead_percent", float64(routingLatency.Microseconds())/float64(totalLatency.Microseconds())*100,
		"key_verify_overhead_percent", float64(keyVerifyLatency.Microseconds())/float64(totalLatency.Microseconds())*100,
	)

	return nil
}
