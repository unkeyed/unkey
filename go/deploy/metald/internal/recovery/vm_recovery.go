package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/health"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// OrphanedVM represents a VM that has lost its process
type OrphanedVM struct {
	VMId       string            `json:"vm_id"`
	ProcessID  string            `json:"process_id"`
	LastSeen   time.Time         `json:"last_seen"`
	Reason     OrphanedReason    `json:"reason"`
	State      metaldv1.VmState  `json:"state"`
	Config     *metaldv1.VmConfig `json:"config,omitempty"`
	DetectedAt time.Time         `json:"detected_at"`
}

type OrphanedReason string

const (
	ReasonProcessDead       OrphanedReason = "process_dead"
	ReasonSocketUnreachable OrphanedReason = "socket_unreachable"
	ReasonHealthCheckFailed OrphanedReason = "health_check_failed"
	ReasonProcessMismatch   OrphanedReason = "process_mismatch"
)

// RecoveryConfig configures VM recovery behavior
type RecoveryConfig struct {
	MaxRetries           int           `json:"max_retries"`            // Maximum recovery attempts (default: 3)
	RetryInterval        time.Duration `json:"retry_interval"`         // Base interval between retries (default: 30s)
	BackoffFactor        float64       `json:"backoff_factor"`         // Exponential backoff multiplier (default: 2.0)
	MaxRetryInterval     time.Duration `json:"max_retry_interval"`     // Maximum retry interval (default: 5m)
	RecoveryTimeout      time.Duration `json:"recovery_timeout"`       // Total time to give up (default: 10m)
	DetectionInterval    time.Duration `json:"detection_interval"`     // How often to scan for orphans (default: 60s)
	Enabled              bool          `json:"enabled"`                // Enable/disable recovery (default: true)
	AllowDataLoss        bool          `json:"allow_data_loss"`        // Allow recovery even if VM state might be lost
}

// DefaultRecoveryConfig returns sensible defaults
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		MaxRetries:        3,
		RetryInterval:     30 * time.Second,
		BackoffFactor:     2.0,
		MaxRetryInterval:  5 * time.Minute,
		RecoveryTimeout:   10 * time.Minute,
		DetectionInterval: 60 * time.Second,
		Enabled:           true,
		AllowDataLoss:     false,
	}
}

// RecoveryAttempt tracks a single recovery attempt
type RecoveryAttempt struct {
	VMId         string    `json:"vm_id"`
	AttemptNum   int       `json:"attempt_num"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	Success      bool      `json:"success"`
	ErrorMsg     string    `json:"error_msg,omitempty"`
	RecoveryType string    `json:"recovery_type"`
}

// VMRecoveryManager manages detection and recovery of orphaned VMs
type VMRecoveryManager struct {
	logger         *slog.Logger
	config         *RecoveryConfig
	healthChecker  *health.VMHealthChecker
	
	// Metrics
	meter                   metric.Meter
	orphanedVMsDetected     metric.Int64Counter
	recoveryAttempts        metric.Int64Counter
	recoverySuccess         metric.Int64Counter
	recoveryDuration        metric.Float64Histogram
	
	// State tracking
	mu                sync.RWMutex
	orphanedVMs       map[string]*OrphanedVM        // vmID -> orphaned VM
	recoveryAttempts  map[string][]*RecoveryAttempt // vmID -> attempts
	activeRecoveries  map[string]context.CancelFunc // vmID -> cancel func
	
	// Interfaces (to be injected)
	processManager VMProcessManager
	vmRegistry     VMRegistry
	
	// Control
	stopCh  chan struct{}
	stopped bool
}

// VMProcessManager interface for process management operations
type VMProcessManager interface {
	GetOrCreateProcess(ctx context.Context, vmID string) (ProcessInfo, error)
	ReleaseProcess(ctx context.Context, vmID string) error
	GetProcessInfo() map[string]ProcessInfo
	IsProcessHealthy(processID string) bool
}

// VMRegistry interface for VM registry operations  
type VMRegistry interface {
	GetVM(vmID string) (VMInfo, bool)
	UpdateVMProcess(vmID, processID string) error
	GetAllVMs() map[string]VMInfo
	MarkVMFailed(vmID string, reason string) error
}

// ProcessInfo represents process information
type ProcessInfo interface {
	GetID() string
	GetSocketPath() string
	GetPID() int
	GetVMID() string
	GetStatus() string
	IsRunning() bool
}

// VMInfo represents VM information from registry
type VMInfo interface {
	GetID() string
	GetProcessID() string
	GetConfig() *metaldv1.VmConfig
	GetState() metaldv1.VmState
	GetLastActivity() time.Time
}

// VMRecreator interface for recreating VMs
type VMRecreator interface {
	RecreateVM(ctx context.Context, vmID string, config *metaldv1.VmConfig) error
}

// NewVMRecoveryManager creates a new VM recovery manager
func NewVMRecoveryManager(
	logger *slog.Logger,
	config *RecoveryConfig,
	healthChecker *health.VMHealthChecker,
) (*VMRecoveryManager, error) {
	if config == nil {
		config = DefaultRecoveryConfig()
	}
	
	// Initialize metrics
	meter := otel.Meter("unkey.metald.vm.recovery")
	
	orphanedVMsDetected, err := meter.Int64Counter(
		"unkey_metald_vm_orphaned_total",
		metric.WithDescription("Total number of orphaned VMs detected"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create orphaned VMs counter: %w", err)
	}
	
	recoveryAttempts, err := meter.Int64Counter(
		"unkey_metald_vm_recovery_attempts_total",
		metric.WithDescription("Total number of VM recovery attempts"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery attempts counter: %w", err)
	}
	
	recoverySuccess, err := meter.Int64Counter(
		"unkey_metald_vm_recovery_success_total",
		metric.WithDescription("Total number of successful VM recoveries"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery success counter: %w", err)
	}
	
	recoveryDuration, err := meter.Float64Histogram(
		"unkey_metald_vm_recovery_duration_seconds",
		metric.WithDescription("Duration of VM recovery operations in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery duration histogram: %w", err)
	}
	
	rm := &VMRecoveryManager{
		logger:              logger.With("component", "vm_recovery_manager"),
		config:              config,
		healthChecker:       healthChecker,
		meter:               meter,
		orphanedVMsDetected: orphanedVMsDetected,
		recoveryAttempts:    recoveryAttempts,
		recoverySuccess:     recoverySuccess,
		recoveryDuration:    recoveryDuration,
		orphanedVMs:         make(map[string]*OrphanedVM),
		recoveryAttempts:    make(map[string][]*RecoveryAttempt),
		activeRecoveries:    make(map[string]context.CancelFunc),
		stopCh:              make(chan struct{}),
	}
	
	// Set up health check callbacks
	if healthChecker != nil {
		healthChecker.SetCallbacks(rm.onVMUnhealthy, rm.onVMRecovered)
	}
	
	return rm, nil
}

// SetProcessManager injects the process manager dependency
func (rm *VMRecoveryManager) SetProcessManager(pm VMProcessManager) {
	rm.processManager = pm
}

// SetVMRegistry injects the VM registry dependency
func (rm *VMRecoveryManager) SetVMRegistry(vr VMRegistry) {
	rm.vmRegistry = vr
}

// Start begins the orphaned VM detection and recovery loops
func (rm *VMRecoveryManager) Start(ctx context.Context) error {
	if !rm.config.Enabled {
		rm.logger.Info("vm recovery manager disabled")
		return nil
	}
	
	if rm.processManager == nil || rm.vmRegistry == nil {
		return fmt.Errorf("process manager and VM registry must be set before starting")
	}
	
	rm.logger.Info("starting vm recovery manager",
		"detection_interval", rm.config.DetectionInterval,
		"max_retries", rm.config.MaxRetries,
		"recovery_timeout", rm.config.RecoveryTimeout,
	)
	
	// Start orphaned VM detection loop
	go rm.detectionLoop(ctx)
	
	return nil
}

// Stop stops the recovery manager
func (rm *VMRecoveryManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.stopped {
		return
	}
	
	rm.logger.Info("stopping vm recovery manager")
	
	// Signal stop
	close(rm.stopCh)
	rm.stopped = true
	
	// Cancel active recoveries
	for vmID, cancel := range rm.activeRecoveries {
		rm.logger.Info("cancelling active recovery", "vm_id", vmID)
		cancel()
	}
	
	rm.logger.Info("vm recovery manager stopped")
}

// detectionLoop periodically scans for orphaned VMs
func (rm *VMRecoveryManager) detectionLoop(ctx context.Context) {
	ticker := time.NewTicker(rm.config.DetectionInterval)
	defer ticker.Stop()
	
	// Perform initial scan
	rm.detectOrphanedVMs(ctx)
	
	for {
		select {
		case <-ctx.Done():
			rm.logger.Debug("detection loop stopped due to context cancellation")
			return
		case <-rm.stopCh:
			rm.logger.Debug("detection loop stopped")
			return
		case <-ticker.C:
			rm.detectOrphanedVMs(ctx)
		}
	}
}

// detectOrphanedVMs scans for orphaned VMs and initiates recovery
func (rm *VMRecoveryManager) detectOrphanedVMs(ctx context.Context) {
	tracer := otel.Tracer("unkey.metald.vm.recovery")
	ctx, span := tracer.Start(ctx, "detect_orphaned_vms")
	defer span.End()
	
	start := time.Now()
	
	// Get all VMs from registry
	allVMs := rm.vmRegistry.GetAllVMs()
	
	// Get all processes from process manager
	allProcesses := rm.processManager.GetProcessInfo()
	
	orphansDetected := 0
	
	for vmID, vmInfo := range allVMs {
		orphan := rm.checkForOrphan(vmInfo, allProcesses)
		if orphan != nil {
			rm.handleOrphanedVM(ctx, orphan)
			orphansDetected++
		}
	}
	
	duration := time.Since(start)
	
	if orphansDetected > 0 {
		rm.logger.Warn("orphaned vms detected",
			"count", orphansDetected,
			"total_vms", len(allVMs),
			"scan_duration", duration,
		)
	} else {
		rm.logger.Debug("orphan detection completed",
			"total_vms", len(allVMs),
			"scan_duration", duration,
		)
	}
	
	span.SetAttributes(
		attribute.Int("total_vms", len(allVMs)),
		attribute.Int("orphans_detected", orphansDetected),
		attribute.Float64("scan_duration_seconds", duration.Seconds()),
	)
}

// checkForOrphan determines if a VM is orphaned
func (rm *VMRecoveryManager) checkForOrphan(vmInfo VMInfo, allProcesses map[string]ProcessInfo) *OrphanedVM {
	vmID := vmInfo.GetID()
	processID := vmInfo.GetProcessID()
	
	// Skip if already being recovered
	rm.mu.RLock()
	if _, recovering := rm.activeRecoveries[vmID]; recovering {
		rm.mu.RUnlock()
		return nil
	}
	rm.mu.RUnlock()
	
	// Check if VM has an assigned process
	if processID == "" {
		return &OrphanedVM{
			VMId:       vmID,
			ProcessID:  processID,
			LastSeen:   vmInfo.GetLastActivity(),
			Reason:     ReasonProcessMismatch,
			State:      vmInfo.GetState(),
			Config:     vmInfo.GetConfig(),
			DetectedAt: time.Now(),
		}
	}
	
	// Check if the assigned process exists
	process, processExists := allProcesses[processID]
	if !processExists {
		return &OrphanedVM{
			VMId:       vmID,
			ProcessID:  processID,
			LastSeen:   vmInfo.GetLastActivity(),
			Reason:     ReasonProcessDead,
			State:      vmInfo.GetState(),
			Config:     vmInfo.GetConfig(),
			DetectedAt: time.Now(),
		}
	}
	
	// Check if process is assigned to this VM
	if process.GetVMID() != vmID {
		return &OrphanedVM{
			VMId:       vmID,
			ProcessID:  processID,
			LastSeen:   vmInfo.GetLastActivity(),
			Reason:     ReasonProcessMismatch,
			State:      vmInfo.GetState(),
			Config:     vmInfo.GetConfig(),
			DetectedAt: time.Now(),
		}
	}
	
	// Check if process is healthy
	if !rm.processManager.IsProcessHealthy(processID) {
		return &OrphanedVM{
			VMId:       vmID,
			ProcessID:  processID,
			LastSeen:   vmInfo.GetLastActivity(),
			Reason:     ReasonSocketUnreachable,
			State:      vmInfo.GetState(),
			Config:     vmInfo.GetConfig(),
			DetectedAt: time.Now(),
		}
	}
	
	// Check health checker status
	if rm.healthChecker != nil {
		if healthStatus, exists := rm.healthChecker.GetVMHealth(vmID); exists && !healthStatus.IsHealthy {
			return &OrphanedVM{
				VMId:       vmID,
				ProcessID:  processID,
				LastSeen:   healthStatus.LastHealthy,
				Reason:     ReasonHealthCheckFailed,
				State:      vmInfo.GetState(),
				Config:     vmInfo.GetConfig(),
				DetectedAt: time.Now(),
			}
		}
	}
	
	return nil
}

// handleOrphanedVM handles detection of an orphaned VM
func (rm *VMRecoveryManager) handleOrphanedVM(ctx context.Context, orphan *OrphanedVM) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Check if we already know about this orphan
	if existing, exists := rm.orphanedVMs[orphan.VMId]; exists {
		// Update the orphan info but don't start new recovery
		existing.LastSeen = orphan.LastSeen
		existing.Reason = orphan.Reason
		return
	}
	
	// Record new orphan
	rm.orphanedVMs[orphan.VMId] = orphan
	
	// Record metric
	rm.orphanedVMsDetected.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("vm_id", orphan.VMId),
			attribute.String("reason", string(orphan.Reason)),
		),
	)
	
	rm.logger.Error("orphaned vm detected",
		"vm_id", orphan.VMId,
		"process_id", orphan.ProcessID,
		"reason", orphan.Reason,
		"last_seen", orphan.LastSeen,
		"downtime", time.Since(orphan.LastSeen),
	)
	
	// Start recovery in background
	go rm.startRecovery(context.Background(), orphan)
}

// startRecovery initiates recovery for an orphaned VM
func (rm *VMRecoveryManager) startRecovery(ctx context.Context, orphan *OrphanedVM) {
	recoveryCtx, cancel := context.WithTimeout(ctx, rm.config.RecoveryTimeout)
	defer cancel()
	
	rm.mu.Lock()
	rm.activeRecoveries[orphan.VMId] = cancel
	rm.mu.Unlock()
	
	defer func() {
		rm.mu.Lock()
		delete(rm.activeRecoveries, orphan.VMId)
		rm.mu.Unlock()
	}()
	
	rm.logger.Info("starting vm recovery",
		"vm_id", orphan.VMId,
		"reason", orphan.Reason,
		"max_retries", rm.config.MaxRetries,
	)
	
	success := rm.performRecovery(recoveryCtx, orphan)
	
	rm.mu.Lock()
	if success {
		// Remove from orphaned list on success
		delete(rm.orphanedVMs, orphan.VMId)
		rm.logger.Info("vm recovery completed successfully", "vm_id", orphan.VMId)
	} else {
		rm.logger.Error("vm recovery failed completely", "vm_id", orphan.VMId)
		// Keep in orphaned list for future attempts or manual intervention
	}
	rm.mu.Unlock()
}

// performRecovery performs the actual recovery with retries
func (rm *VMRecoveryManager) performRecovery(ctx context.Context, orphan *OrphanedVM) bool {
	start := time.Now()
	
	for attempt := 1; attempt <= rm.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			rm.logger.Warn("recovery cancelled", "vm_id", orphan.VMId, "attempt", attempt)
			return false
		default:
		}
		
		attemptStart := time.Now()
		success, err := rm.attemptRecovery(ctx, orphan, attempt)
		attemptDuration := time.Since(attemptStart)
		
		// Record attempt
		recoveryAttempt := &RecoveryAttempt{
			VMId:         orphan.VMId,
			AttemptNum:   attempt,
			StartedAt:    attemptStart,
			CompletedAt:  time.Now(),
			Success:      success,
			RecoveryType: string(orphan.Reason),
		}
		
		if err != nil {
			recoveryAttempt.ErrorMsg = err.Error()
		}
		
		rm.recordRecoveryAttempt(orphan.VMId, recoveryAttempt)
		
		// Record metrics
		rm.recoveryAttempts.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("vm_id", orphan.VMId),
				attribute.String("reason", string(orphan.Reason)),
				attribute.Int("attempt", attempt),
				attribute.Bool("success", success),
			),
		)
		
		if success {
			rm.recoverySuccess.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("vm_id", orphan.VMId),
					attribute.String("reason", string(orphan.Reason)),
					attribute.Int("total_attempts", attempt),
				),
			)
			
			totalDuration := time.Since(start)
			rm.recoveryDuration.Record(ctx, totalDuration.Seconds(),
				metric.WithAttributes(
					attribute.String("vm_id", orphan.VMId),
					attribute.String("outcome", "success"),
				),
			)
			
			rm.logger.Info("vm recovery successful",
				"vm_id", orphan.VMId,
				"attempt", attempt,
				"total_duration", totalDuration,
				"attempt_duration", attemptDuration,
			)
			
			return true
		}
		
		rm.logger.Warn("vm recovery attempt failed",
			"vm_id", orphan.VMId,
			"attempt", attempt,
			"max_retries", rm.config.MaxRetries,
			"error", err,
			"duration", attemptDuration,
		)
		
		// Wait before retry (unless it's the last attempt)
		if attempt < rm.config.MaxRetries {
			retryDelay := rm.calculateRetryDelay(attempt)
			rm.logger.Debug("waiting before retry",
				"vm_id", orphan.VMId,
				"retry_delay", retryDelay,
			)
			
			select {
			case <-ctx.Done():
				return false
			case <-time.After(retryDelay):
				// Continue to next attempt
			}
		}
	}
	
	// All attempts failed
	totalDuration := time.Since(start)
	rm.recoveryDuration.Record(ctx, totalDuration.Seconds(),
		metric.WithAttributes(
			attribute.String("vm_id", orphan.VMId),
			attribute.String("outcome", "failure"),
		),
	)
	
	return false
}

// attemptRecovery performs a single recovery attempt
func (rm *VMRecoveryManager) attemptRecovery(ctx context.Context, orphan *OrphanedVM, attempt int) (bool, error) {
	tracer := otel.Tracer("unkey.metald.vm.recovery")
	ctx, span := tracer.Start(ctx, "recovery_attempt",
		trace.WithAttributes(
			attribute.String("vm_id", orphan.VMId),
			attribute.String("reason", string(orphan.Reason)),
			attribute.Int("attempt", attempt),
		),
	)
	defer span.End()
	
	rm.logger.Info("attempting vm recovery",
		"vm_id", orphan.VMId,
		"attempt", attempt,
		"reason", orphan.Reason,
	)
	
	// Step 1: Clean up old process if it exists
	if orphan.ProcessID != "" {
		if err := rm.processManager.ReleaseProcess(ctx, orphan.VMId); err != nil {
			rm.logger.Warn("failed to release old process",
				"vm_id", orphan.VMId,
				"process_id", orphan.ProcessID,
				"error", err,
			)
			// Continue anyway - process might already be gone
		}
	}
	
	// Step 2: Create new process
	newProcess, err := rm.processManager.GetOrCreateProcess(ctx, orphan.VMId)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to create new process: %w", err)
	}
	
	// Step 3: Update VM registry with new process
	if err := rm.vmRegistry.UpdateVMProcess(orphan.VMId, newProcess.GetID()); err != nil {
		// Try to clean up the process we just created
		rm.processManager.ReleaseProcess(ctx, orphan.VMId)
		span.RecordError(err)
		return false, fmt.Errorf("failed to update vm registry: %w", err)
	}
	
	// Step 4: Restart health monitoring if available
	if rm.healthChecker != nil {
		if err := rm.healthChecker.StartMonitoring(
			orphan.VMId,
			newProcess.GetID(),
			newProcess.GetSocketPath(),
			newProcess.GetPID(),
		); err != nil {
			rm.logger.Warn("failed to restart health monitoring",
				"vm_id", orphan.VMId,
				"error", err,
			)
			// Don't fail recovery for this
		}
	}
	
	span.SetAttributes(
		attribute.String("new_process_id", newProcess.GetID()),
		attribute.String("new_socket_path", newProcess.GetSocketPath()),
	)
	
	rm.logger.Info("vm recovery attempt completed",
		"vm_id", orphan.VMId,
		"attempt", attempt,
		"new_process_id", newProcess.GetID(),
	)
	
	return true, nil
}

// calculateRetryDelay calculates the delay before the next retry attempt
func (rm *VMRecoveryManager) calculateRetryDelay(attempt int) time.Duration {
	baseDelay := rm.config.RetryInterval
	
	// Apply exponential backoff
	backoffMultiplier := 1.0
	for i := 1; i < attempt; i++ {
		backoffMultiplier *= rm.config.BackoffFactor
	}
	
	delay := time.Duration(float64(baseDelay) * backoffMultiplier)
	
	// Cap at maximum retry interval
	if delay > rm.config.MaxRetryInterval {
		delay = rm.config.MaxRetryInterval
	}
	
	return delay
}

// recordRecoveryAttempt records a recovery attempt
func (rm *VMRecoveryManager) recordRecoveryAttempt(vmID string, attempt *RecoveryAttempt) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.recoveryAttempts[vmID] == nil {
		rm.recoveryAttempts[vmID] = make([]*RecoveryAttempt, 0)
	}
	
	rm.recoveryAttempts[vmID] = append(rm.recoveryAttempts[vmID], attempt)
	
	// Keep only the last 10 attempts per VM to prevent memory growth
	if len(rm.recoveryAttempts[vmID]) > 10 {
		rm.recoveryAttempts[vmID] = rm.recoveryAttempts[vmID][1:]
	}
}

// onVMUnhealthy is called when a VM becomes unhealthy
func (rm *VMRecoveryManager) onVMUnhealthy(vmID string, status *health.VMHealthStatus) {
	rm.logger.Warn("vm marked unhealthy by health checker",
		"vm_id", vmID,
		"failure_count", status.FailureCount,
		"error", status.ErrorMsg,
	)
	
	// The detection loop will pick this up and handle it
}

// onVMRecovered is called when a VM recovers
func (rm *VMRecoveryManager) onVMRecovered(vmID string, status *health.VMHealthStatus) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Remove from orphaned list if it was there
	if _, wasOrphaned := rm.orphanedVMs[vmID]; wasOrphaned {
		delete(rm.orphanedVMs, vmID)
		rm.logger.Info("vm removed from orphaned list due to recovery",
			"vm_id", vmID,
		)
	}
}

// GetOrphanedVMs returns the current list of orphaned VMs
func (rm *VMRecoveryManager) GetOrphanedVMs() map[string]*OrphanedVM {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	result := make(map[string]*OrphanedVM)
	for vmID, orphan := range rm.orphanedVMs {
		orphanCopy := *orphan
		result[vmID] = &orphanCopy
	}
	
	return result
}

// GetRecoveryAttempts returns recovery attempts for a VM
func (rm *VMRecoveryManager) GetRecoveryAttempts(vmID string) []*RecoveryAttempt {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	attempts := rm.recoveryAttempts[vmID]
	if attempts == nil {
		return nil
	}
	
	// Return copy to avoid race conditions
	result := make([]*RecoveryAttempt, len(attempts))
	for i, attempt := range attempts {
		attemptCopy := *attempt
		result[i] = &attemptCopy
	}
	
	return result
}