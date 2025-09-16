package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
)

// Health status constants
const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusDegraded  = "degraded"
)

// Handler provides health check endpoints
type Handler struct {
	backend   types.Backend
	logger    *slog.Logger
	startTime time.Time
}

// NewHandler creates a new health check handler
func NewHandler(backend types.Backend, logger *slog.Logger, startTime time.Time) *Handler {
	return &Handler{
		backend:   backend,
		logger:    logger.With("component", "health"),
		startTime: startTime,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Version   string           `json:"version"`
	Backend   BackendHealth    `json:"backend"`
	Checks    map[string]Check `json:"checks"`
}

// BackendHealth contains backend-specific health information
type BackendHealth struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Check represents an individual health check result
type Check struct {
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration_ms"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// ServeHTTP handles health check requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	h.logger.LogAttrs(ctx, slog.LevelInfo, "health check requested",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.String("user_agent", r.Header.Get("User-Agent")),
	)

	// Perform health checks
	response := h.performHealthChecks(ctx)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Determine HTTP status code based on overall health
	statusCode := http.StatusOK
	if response.Status != StatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.LogAttrs(ctx, slog.LevelError, "failed to encode health response",
			slog.String("error", err.Error()),
		)
		return
	}

	duration := time.Since(startTime)
	h.logger.LogAttrs(ctx, slog.LevelInfo, "health check completed",
		slog.String("status", response.Status),
		slog.Duration("duration", duration),
		slog.Int("status_code", statusCode),
	)
}

// performHealthChecks executes all health checks and returns the result
func (h *Handler) performHealthChecks(ctx context.Context) *HealthResponse {
	timestamp := time.Now()
	checks := make(map[string]Check)

	// Check backend health
	backendHealth := h.checkBackendHealth(ctx, checks)

	//exhaustruct:ignore
	checks["system_info"] = Check{
		Status:    StatusHealthy,
		Timestamp: timestamp,
	}

	// Determine overall status
	overallStatus := StatusHealthy
	if backendHealth.Status != StatusHealthy {
		overallStatus = StatusUnhealthy
	}

	for _, check := range checks {
		if check.Status != StatusHealthy {
			overallStatus = StatusDegraded
			break
		}
	}

	return &HealthResponse{
		Status:    overallStatus,
		Timestamp: timestamp,
		Version:   "dev", // AIDEV-TODO: Get from build info
		Backend:   backendHealth,
		Checks:    checks,
	}
}

// checkBackendHealth checks the health of the hypervisor backend
func (h *Handler) checkBackendHealth(ctx context.Context, checks map[string]Check) BackendHealth {
	checkStart := time.Now()

	// Create a timeout context for the backend ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := h.backend.Ping(pingCtx)
	duration := time.Since(checkStart)

	backendHealth := BackendHealth{ //nolint:exhaustruct // Status and Error fields are set conditionally based on backend response
		Type: "firecracker", // Only Firecracker is supported
	}

	if err != nil {
		h.logger.LogAttrs(ctx, slog.LevelWarn, "backend ping failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", duration),
		)

		backendHealth.Status = StatusUnhealthy
		backendHealth.Error = err.Error()

		checks["backend_ping"] = Check{
			Status:    StatusUnhealthy,
			Duration:  duration,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
	} else {
		h.logger.LogAttrs(ctx, slog.LevelDebug, "backend ping successful",
			slog.Duration("duration", duration),
		)

		backendHealth.Status = StatusHealthy

		checks["backend_ping"] = Check{ //nolint:exhaustruct // Error field is set conditionally based on check result
			Status:    StatusHealthy,
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return backendHealth
}
