package health

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// VMHealthStatus represents the health status of a VM
type VMHealthStatus struct {
	VMId         string    `json:"vm_id"`
	ProcessID    string    `json:"process_id"`
	IsHealthy    bool      `json:"is_healthy"`
	LastCheck    time.Time `json:"last_check"`
	LastHealthy  time.Time `json:"last_healthy"`
	ProcessPID   int       `json:"process_pid"`
	SocketPath   string    `json:"socket_path"`
	ErrorMsg     string    `json:"error_msg,omitempty"`
	CheckCount   int64     `json:"check_count"`
	FailureCount int64     `json:"failure_count"`
}

// HealthCheckConfig configures VM health checking behavior
type HealthCheckConfig struct {
	Interval          time.Duration `json:"interval"`           // How often to check (default: 30s)
	Timeout           time.Duration `json:"timeout"`            // Per-check timeout (default: 5s)
	FailureThreshold  int           `json:"failure_threshold"`  // Consecutive failures before unhealthy (default: 3)
	RecoveryThreshold int           `json:"recovery_threshold"` // Consecutive successes before healthy (default: 2)
	Enabled           bool          `json:"enabled"`            // Enable/disable health checking
}

// DefaultHealthCheckConfig returns sensible defaults
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Interval:          30 * time.Second,
		Timeout:           5 * time.Second,
		FailureThreshold:  3,
		RecoveryThreshold: 2,
		Enabled:           true,
	}
}

// VMHealthChecker manages health checking for VMs
type VMHealthChecker struct {
	logger     *slog.Logger
	config     *HealthCheckConfig
	httpClient *http.Client

	// Metrics
	meter               metric.Meter
	healthCheckTotal    metric.Int64Counter
	healthCheckFailed   metric.Int64Counter
	healthCheckDuration metric.Float64Histogram

	// State tracking
	mu           sync.RWMutex
	vmStatus     map[string]*VMHealthStatus    // vmID -> status
	activeChecks map[string]context.CancelFunc // vmID -> cancel function

	// Callbacks
	onVMUnhealthy func(vmID string, status *VMHealthStatus)
	onVMRecovered func(vmID string, status *VMHealthStatus)
}

// NewVMHealthChecker creates a new VM health checker
func NewVMHealthChecker(logger *slog.Logger, config *HealthCheckConfig) (*VMHealthChecker, error) {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	// Create HTTP client with Unix socket transport
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// This will be overridden per-request for different socket paths
				return nil, fmt.Errorf("socket not configured")
			},
		},
	}

	// Initialize metrics
	meter := otel.Meter("unkey.metald.vm.health")

	healthCheckTotal, err := meter.Int64Counter(
		"unkey_metald_vm_health_checks_total",
		metric.WithDescription("Total number of VM health checks performed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create health check counter: %w", err)
	}

	healthCheckFailed, err := meter.Int64Counter(
		"unkey_metald_vm_health_check_failures_total",
		metric.WithDescription("Total number of failed VM health checks"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create health check failure counter: %w", err)
	}

	healthCheckDuration, err := meter.Float64Histogram(
		"unkey_metald_vm_health_check_duration_seconds",
		metric.WithDescription("Duration of VM health checks in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create health check duration histogram: %w", err)
	}

	//exhaustruct:ignore
	return &VMHealthChecker{
		logger:              logger.With("component", "vm_health_checker"),
		config:              config,
		httpClient:          httpClient,
		meter:               meter,
		healthCheckTotal:    healthCheckTotal,
		healthCheckFailed:   healthCheckFailed,
		healthCheckDuration: healthCheckDuration,
		vmStatus:            make(map[string]*VMHealthStatus),
		activeChecks:        make(map[string]context.CancelFunc),
	}, nil
}

// SetCallbacks sets callback functions for health state changes
func (hc *VMHealthChecker) SetCallbacks(
	onUnhealthy func(vmID string, status *VMHealthStatus),
	onRecovered func(vmID string, status *VMHealthStatus),
) {
	hc.onVMUnhealthy = onUnhealthy
	hc.onVMRecovered = onRecovered
}

// StartMonitoring begins health checking for a VM
func (hc *VMHealthChecker) StartMonitoring(vmID, processID, socketPath string, processPID int) error {
	if !hc.config.Enabled {
		hc.logger.Debug("health checking disabled", "vm_id", vmID)
		return nil
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Stop existing monitoring if any
	if cancel, exists := hc.activeChecks[vmID]; exists {
		cancel()
		delete(hc.activeChecks, vmID)
	}

	// Initialize status
	//exhaustruct:ignore
	status := &VMHealthStatus{
		VMId:         vmID,
		ProcessID:    processID,
		IsHealthy:    true, // Assume healthy initially
		LastCheck:    time.Now(),
		LastHealthy:  time.Now(),
		ProcessPID:   processPID,
		SocketPath:   socketPath,
		CheckCount:   0,
		FailureCount: 0,
	}
	hc.vmStatus[vmID] = status

	// Start monitoring goroutine
	ctx, cancel := context.WithCancel(context.Background())
	hc.activeChecks[vmID] = cancel

	go hc.monitorVM(ctx, vmID)

	hc.logger.Info("started vm health monitoring",
		"vm_id", vmID,
		"process_id", processID,
		"socket_path", socketPath,
		"interval", hc.config.Interval,
	)

	return nil
}

// StopMonitoring stops health checking for a VM
func (hc *VMHealthChecker) StopMonitoring(vmID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if cancel, exists := hc.activeChecks[vmID]; exists {
		cancel()
		delete(hc.activeChecks, vmID)
	}

	delete(hc.vmStatus, vmID)

	hc.logger.Info("stopped vm health monitoring", "vm_id", vmID)
}

// GetVMHealth returns the current health status of a VM
func (hc *VMHealthChecker) GetVMHealth(vmID string) (*VMHealthStatus, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status, exists := hc.vmStatus[vmID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	statusCopy := *status
	return &statusCopy, true
}

// GetAllVMHealth returns health status for all monitored VMs
func (hc *VMHealthChecker) GetAllVMHealth() map[string]*VMHealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]*VMHealthStatus)
	for vmID, status := range hc.vmStatus {
		statusCopy := *status
		result[vmID] = &statusCopy
	}

	return result
}

// monitorVM runs the health checking loop for a single VM
func (hc *VMHealthChecker) monitorVM(ctx context.Context, vmID string) {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	// Perform initial check immediately
	hc.performHealthCheck(ctx, vmID)

	for {
		select {
		case <-ctx.Done():
			hc.logger.DebugContext(ctx, "health monitoring stopped", "vm_id", vmID)
			return
		case <-ticker.C:
			hc.performHealthCheck(ctx, vmID)
		}
	}
}

// performHealthCheck performs a single health check for a VM
func (hc *VMHealthChecker) performHealthCheck(ctx context.Context, vmID string) {
	start := time.Now()

	// Create trace span for observability
	tracer := otel.Tracer("unkey.metald.vm.health")
	ctx, span := tracer.Start(ctx, "vm_health_check",
		trace.WithAttributes(
			attribute.String("vm_id", vmID),
		),
	)
	defer span.End()

	hc.mu.Lock()
	status, exists := hc.vmStatus[vmID]
	if !exists {
		hc.mu.Unlock()
		return
	}

	// Create local copy for thread safety
	socketPath := status.SocketPath
	processPID := status.ProcessPID
	hc.mu.Unlock()

	// Perform the actual health check
	checkCtx, cancel := context.WithTimeout(ctx, hc.config.Timeout)
	defer cancel()

	isHealthy, errorMsg := hc.checkVMHealth(checkCtx, socketPath, processPID)
	duration := time.Since(start)

	// Record metrics
	hc.healthCheckTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("vm_id", vmID),
			attribute.Bool("healthy", isHealthy),
		),
	)

	if !isHealthy {
		hc.healthCheckFailed.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("vm_id", vmID),
				attribute.String("error", errorMsg),
			),
		)
	}

	hc.healthCheckDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("vm_id", vmID),
		),
	)

	// Update status and check for state transitions
	hc.updateVMHealthStatus(vmID, isHealthy, errorMsg, duration)

	span.SetAttributes(
		attribute.Bool("healthy", isHealthy),
		attribute.Float64("duration_seconds", duration.Seconds()),
	)

	if !isHealthy {
		span.RecordError(fmt.Errorf("health check failed: %s", errorMsg))
	}
}

// checkVMHealth performs the actual health check logic
func (hc *VMHealthChecker) checkVMHealth(ctx context.Context, socketPath string, processPID int) (bool, string) {
	// 1. Check if socket file exists
	if _, err := os.Stat(socketPath); err != nil {
		return false, fmt.Sprintf("socket file missing: %v", err)
	}

	// 2. Check if process is still running
	if !hc.isProcessRunning(processPID) {
		return false, "process not running"
	}

	// 3. Test socket connectivity
	conn, err := net.DialTimeout("unix", socketPath, hc.config.Timeout)
	if err != nil {
		return false, fmt.Sprintf("socket unreachable: %v", err)
	}
	defer conn.Close()

	// 4. Test Firecracker API endpoint
	client := &http.Client{
		Timeout: hc.config.Timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/", nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("api request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return false, fmt.Sprintf("api error: status %d", resp.StatusCode)
	}

	return true, ""
}

// updateVMHealthStatus updates the health status and handles state transitions
func (hc *VMHealthChecker) updateVMHealthStatus(vmID string, isHealthy bool, errorMsg string, duration time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	status, exists := hc.vmStatus[vmID]
	if !exists {
		return
	}

	now := time.Now()
	wasHealthy := status.IsHealthy

	// Update basic status
	status.LastCheck = now
	status.CheckCount++

	if isHealthy {
		hc.handleHealthyStatus(status, wasHealthy, vmID, now)
	} else {
		hc.handleUnhealthyStatus(status, wasHealthy, vmID, errorMsg, duration)
	}
}

// isProcessRunning checks if a process is still running
func (hc *VMHealthChecker) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Check if /proc/pid exists
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
		return false
	}

	return true
}

// Shutdown stops all health checking
func (hc *VMHealthChecker) Shutdown() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.logger.Info("shutting down vm health checker")

	// Cancel all active checks
	for vmID, cancel := range hc.activeChecks {
		cancel()
		hc.logger.Debug("stopped health monitoring", "vm_id", vmID)
	}

	// Clear state
	hc.activeChecks = make(map[string]context.CancelFunc)
	hc.vmStatus = make(map[string]*VMHealthStatus)

	hc.logger.Info("vm health checker shutdown complete")
}

// handleHealthyStatus updates status when health check succeeds
func (hc *VMHealthChecker) handleHealthyStatus(status *VMHealthStatus, wasHealthy bool, vmID string, now time.Time) {
	status.LastHealthy = now
	status.ErrorMsg = ""

	// Reset failure count on success
	if status.FailureCount > 0 {
		hc.logger.Debug("vm health check succeeded after failures",
			"vm_id", vmID,
			"previous_failures", status.FailureCount,
		)
	}
	status.FailureCount = 0

	// Check for recovery (unhealthy -> healthy transition)
	if !wasHealthy {
		status.IsHealthy = true
		hc.logger.Info("vm recovered",
			"vm_id", vmID,
			"downtime", now.Sub(status.LastHealthy),
		)

		// Trigger recovery callback
		if hc.onVMRecovered != nil {
			go hc.onVMRecovered(vmID, status)
		}
	}
}

// handleUnhealthyStatus updates status when health check fails
func (hc *VMHealthChecker) handleUnhealthyStatus(status *VMHealthStatus, wasHealthy bool, vmID string, errorMsg string, duration time.Duration) {
	status.FailureCount++
	status.ErrorMsg = errorMsg

	hc.logger.Warn("vm health check failed",
		"vm_id", vmID,
		"failure_count", status.FailureCount,
		"error", errorMsg,
		"duration", duration,
	)

	// Check if we should mark as unhealthy
	if wasHealthy && status.FailureCount >= int64(hc.config.FailureThreshold) {
		status.IsHealthy = false
		hc.logger.Error("vm marked as unhealthy",
			"vm_id", vmID,
			"consecutive_failures", status.FailureCount,
			"threshold", hc.config.FailureThreshold,
		)

		// Trigger unhealthy callback
		if hc.onVMUnhealthy != nil {
			go hc.onVMUnhealthy(vmID, status)
		}
	}
}
