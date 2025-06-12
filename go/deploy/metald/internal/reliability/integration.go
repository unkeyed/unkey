package reliability

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/health"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/process"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/recovery"
)

// ReliabilityConfig configures the reliability subsystem
type ReliabilityConfig struct {
	Enabled           bool                      `json:"enabled"`
	HealthCheck       *health.HealthCheckConfig `json:"health_check"`
	Recovery          *recovery.RecoveryConfig  `json:"recovery"`
	ReconcileInterval time.Duration             `json:"reconcile_interval"`
}

// DefaultReliabilityConfig returns sensible defaults
func DefaultReliabilityConfig() *ReliabilityConfig {
	return &ReliabilityConfig{
		Enabled:           true,
		HealthCheck:       health.DefaultHealthCheckConfig(),
		Recovery:          recovery.DefaultRecoveryConfig(),
		ReconcileInterval: 5 * time.Minute,
	}
}

// ReliabilityManager integrates health checking and recovery with existing VM management
type ReliabilityManager struct {
	logger          *slog.Logger
	config          *ReliabilityConfig
	healthChecker   *health.VMHealthChecker
	recoveryManager *recovery.VMRecoveryManager
	processManager  *process.Manager

	// VM registry adapter
	vmRegistry *VMRegistryAdapter
}

// VMRegistryAdapter adapts the managed client's VM registry to the recovery interfaces
type VMRegistryAdapter struct {
	managedVMs map[string]*ManagedVMInfo
	logger     *slog.Logger
}

// ManagedVMInfo implements the recovery.VMInfo interface
type ManagedVMInfo struct {
	id           string
	processID    string
	config       *metaldv1.VmConfig
	state        metaldv1.VmState
	lastActivity time.Time
}

func (m *ManagedVMInfo) GetID() string                 { return m.id }
func (m *ManagedVMInfo) GetProcessID() string          { return m.processID }
func (m *ManagedVMInfo) GetConfig() *metaldv1.VmConfig { return m.config }
func (m *ManagedVMInfo) GetState() metaldv1.VmState    { return m.state }
func (m *ManagedVMInfo) GetLastActivity() time.Time    { return m.lastActivity }

// ProcessInfoAdapter adapts the process manager's process info to the recovery interface
type ProcessInfoAdapter struct {
	proc *process.FirecrackerProcess
}

func (p *ProcessInfoAdapter) GetID() string         { return p.proc.ID }
func (p *ProcessInfoAdapter) GetSocketPath() string { return p.proc.SocketPath }
func (p *ProcessInfoAdapter) GetPID() int           { return p.proc.Process.Pid }
func (p *ProcessInfoAdapter) GetVMID() string       { return p.proc.VMID }
func (p *ProcessInfoAdapter) GetStatus() string     { return string(p.proc.Status) }
func (p *ProcessInfoAdapter) IsRunning() bool {
	return p.proc.Status == process.StatusReady || p.proc.Status == process.StatusBusy
}

// ProcessManagerAdapter adapts the process manager to the recovery interface
type ProcessManagerAdapter struct {
	manager *process.Manager
	logger  *slog.Logger
}

func (p *ProcessManagerAdapter) GetOrCreateProcess(ctx context.Context, vmID string) (recovery.ProcessInfo, error) {
	proc, err := p.manager.GetOrCreateProcess(ctx, vmID)
	if err != nil {
		return nil, err
	}
	return &ProcessInfoAdapter{proc: proc}, nil
}

func (p *ProcessManagerAdapter) ReleaseProcess(ctx context.Context, vmID string) error {
	return p.manager.ReleaseProcess(ctx, vmID)
}

func (p *ProcessManagerAdapter) GetProcessInfo() map[string]recovery.ProcessInfo {
	processes := p.manager.GetProcessInfo()
	result := make(map[string]recovery.ProcessInfo)

	for id, proc := range processes {
		result[id] = &ProcessInfoAdapter{proc: proc}
	}

	return result
}

func (p *ProcessManagerAdapter) IsProcessHealthy(processID string) bool {
	processes := p.manager.GetProcessInfo()
	proc, exists := processes[processID]
	if !exists {
		return false
	}

	return proc.Status == process.StatusReady || proc.Status == process.StatusBusy
}

// VMRegistryAdapter methods
func NewVMRegistryAdapter(logger *slog.Logger) *VMRegistryAdapter {
	return &VMRegistryAdapter{
		managedVMs: make(map[string]*ManagedVMInfo),
		logger:     logger.With("component", "vm_registry_adapter"),
	}
}

func (vr *VMRegistryAdapter) GetVM(vmID string) (recovery.VMInfo, bool) {
	vm, exists := vr.managedVMs[vmID]
	return vm, exists
}

func (vr *VMRegistryAdapter) UpdateVMProcess(vmID, processID string) error {
	vm, exists := vr.managedVMs[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found in registry", vmID)
	}

	vm.processID = processID
	vm.lastActivity = time.Now()

	vr.logger.Info("updated vm process mapping",
		"vm_id", vmID,
		"new_process_id", processID,
	)

	return nil
}

func (vr *VMRegistryAdapter) GetAllVMs() map[string]recovery.VMInfo {
	result := make(map[string]recovery.VMInfo)
	for id, vm := range vr.managedVMs {
		result[id] = vm
	}
	return result
}

func (vr *VMRegistryAdapter) MarkVMFailed(vmID string, reason string) error {
	vm, exists := vr.managedVMs[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found in registry", vmID)
	}

	vm.state = metaldv1.VmState_VM_STATE_UNSPECIFIED  // Indicates failed/unknown state
	vm.lastActivity = time.Now()

	vr.logger.Error("marked vm as failed",
		"vm_id", vmID,
		"reason", reason,
	)

	return nil
}

// AddVM adds a VM to the registry
func (vr *VMRegistryAdapter) AddVM(vmID, processID string, config *metaldv1.VmConfig, state metaldv1.VmState) {
	vr.managedVMs[vmID] = &ManagedVMInfo{
		id:           vmID,
		processID:    processID,
		config:       config,
		state:        state,
		lastActivity: time.Now(),
	}

	vr.logger.Info("added vm to registry",
		"vm_id", vmID,
		"process_id", processID,
		"state", state,
	)
}

// RemoveVM removes a VM from the registry
func (vr *VMRegistryAdapter) RemoveVM(vmID string) {
	delete(vr.managedVMs, vmID)
	vr.logger.Info("removed vm from registry", "vm_id", vmID)
}

// UpdateVMState updates the state of a VM
func (vr *VMRegistryAdapter) UpdateVMState(vmID string, state metaldv1.VmState) error {
	vm, exists := vr.managedVMs[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found in registry", vmID)
	}

	vm.state = state
	vm.lastActivity = time.Now()

	vr.logger.Debug("updated vm state",
		"vm_id", vmID,
		"new_state", state,
	)

	return nil
}

// NewReliabilityManager creates a new reliability manager
func NewReliabilityManager(
	logger *slog.Logger,
	config *ReliabilityConfig,
	processManager *process.Manager,
) (*ReliabilityManager, error) {
	if config == nil {
		config = DefaultReliabilityConfig()
	}

	if !config.Enabled {
		logger.Info("reliability subsystem disabled")
		return &ReliabilityManager{
			logger: logger.With("component", "reliability_manager"),
			config: config,
		}, nil
	}

	// Initialize health checker
	healthChecker, err := health.NewVMHealthChecker(logger, config.HealthCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to create health checker: %w", err)
	}

	// Initialize recovery manager
	recoveryManager, err := recovery.NewVMRecoveryManager(logger, config.Recovery, healthChecker)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery manager: %w", err)
	}

	// Create VM registry adapter
	vmRegistry := NewVMRegistryAdapter(logger)

	// Create process manager adapter
	processManagerAdapter := &ProcessManagerAdapter{
		manager: processManager,
		logger:  logger,
	}

	// Wire up dependencies
	recoveryManager.SetProcessManager(processManagerAdapter)
	recoveryManager.SetVMRegistry(vmRegistry)

	return &ReliabilityManager{
		logger:          logger.With("component", "reliability_manager"),
		config:          config,
		healthChecker:   healthChecker,
		recoveryManager: recoveryManager,
		processManager:  processManager,
		vmRegistry:      vmRegistry,
	}, nil
}

// Start starts the reliability subsystem
func (rm *ReliabilityManager) Start(ctx context.Context) error {
	if !rm.config.Enabled {
		rm.logger.Info("reliability subsystem disabled")
		return nil
	}

	rm.logger.Info("starting reliability subsystem")

	// Start recovery manager
	if err := rm.recoveryManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start recovery manager: %w", err)
	}

	rm.logger.Info("reliability subsystem started")
	return nil
}

// Stop stops the reliability subsystem
func (rm *ReliabilityManager) Stop() {
	if !rm.config.Enabled {
		return
	}

	rm.logger.Info("stopping reliability subsystem")

	// Stop recovery manager
	if rm.recoveryManager != nil {
		rm.recoveryManager.Stop()
	}

	// Stop health checker
	if rm.healthChecker != nil {
		rm.healthChecker.Shutdown()
	}

	rm.logger.Info("reliability subsystem stopped")
}

// OnVMCreated should be called when a VM is created
func (rm *ReliabilityManager) OnVMCreated(vmID, processID string, config *metaldv1.VmConfig) {
	if !rm.config.Enabled {
		return
	}

	// Add to registry
	rm.vmRegistry.AddVM(vmID, processID, config, metaldv1.VmState_VM_STATE_CREATED)

	// Start health monitoring if we have process info
	if rm.healthChecker != nil && processID != "" {
		processes := rm.processManager.GetProcessInfo()
		if proc, exists := processes[processID]; exists {
			err := rm.healthChecker.StartMonitoring(vmID, processID, proc.SocketPath, proc.Process.Pid)
			if err != nil {
				rm.logger.Warn("failed to start health monitoring for new vm",
					"vm_id", vmID,
					"process_id", processID,
					"error", err,
				)
			}
		}
	}

	rm.logger.Info("reliability tracking started for vm",
		"vm_id", vmID,
		"process_id", processID,
	)
}

// OnVMStarted should be called when a VM is successfully started
func (rm *ReliabilityManager) OnVMStarted(vmID string) {
	if !rm.config.Enabled {
		return
	}

	rm.vmRegistry.UpdateVMState(vmID, metaldv1.VmState_VM_STATE_RUNNING)

	rm.logger.Debug("vm marked as running",
		"vm_id", vmID,
	)
}

// OnVMStopped should be called when a VM is stopped
func (rm *ReliabilityManager) OnVMStopped(vmID string) {
	if !rm.config.Enabled {
		return
	}

	rm.vmRegistry.UpdateVMState(vmID, metaldv1.VmState_VM_STATE_SHUTDOWN)

	// Stop health monitoring
	if rm.healthChecker != nil {
		rm.healthChecker.StopMonitoring(vmID)
	}

	rm.logger.Debug("vm marked as stopped",
		"vm_id", vmID,
	)
}

// OnVMDeleted should be called when a VM is deleted
func (rm *ReliabilityManager) OnVMDeleted(vmID string) {
	if !rm.config.Enabled {
		return
	}

	// Remove from registry
	rm.vmRegistry.RemoveVM(vmID)

	// Stop health monitoring
	if rm.healthChecker != nil {
		rm.healthChecker.StopMonitoring(vmID)
	}

	rm.logger.Info("reliability tracking stopped for vm",
		"vm_id", vmID,
	)
}

// OnVMError should be called when a VM encounters an error
func (rm *ReliabilityManager) OnVMError(vmID string, err error) {
	if !rm.config.Enabled {
		return
	}

	rm.vmRegistry.MarkVMFailed(vmID, err.Error())

	rm.logger.Error("vm error reported",
		"vm_id", vmID,
		"error", err,
	)
}

// GetVMHealth returns health information for a VM
func (rm *ReliabilityManager) GetVMHealth(vmID string) (*health.VMHealthStatus, bool) {
	if !rm.config.Enabled || rm.healthChecker == nil {
		return nil, false
	}

	return rm.healthChecker.GetVMHealth(vmID)
}

// GetAllVMHealth returns health information for all VMs
func (rm *ReliabilityManager) GetAllVMHealth() map[string]*health.VMHealthStatus {
	if !rm.config.Enabled || rm.healthChecker == nil {
		return make(map[string]*health.VMHealthStatus)
	}

	return rm.healthChecker.GetAllVMHealth()
}

// GetOrphanedVMs returns currently orphaned VMs
func (rm *ReliabilityManager) GetOrphanedVMs() map[string]*recovery.OrphanedVM {
	if !rm.config.Enabled || rm.recoveryManager == nil {
		return make(map[string]*recovery.OrphanedVM)
	}

	return rm.recoveryManager.GetOrphanedVMs()
}

// GetRecoveryAttempts returns recovery attempts for a VM
func (rm *ReliabilityManager) GetRecoveryAttempts(vmID string) []*recovery.RecoveryAttempt {
	if !rm.config.Enabled || rm.recoveryManager == nil {
		return nil
	}

	return rm.recoveryManager.GetRecoveryAttempts(vmID)
}

// ForceRecovery manually triggers recovery for a VM via logging
// Operators can trigger recovery by restarting the service or using SIGUSR1
func (rm *ReliabilityManager) ForceRecovery(ctx context.Context, vmID string) error {
	if !rm.config.Enabled {
		return fmt.Errorf("reliability subsystem disabled")
	}

	rm.logger.Error("manual recovery requested - restart service or wait for next detection cycle",
		"vm_id", vmID,
		"action_required", "service_restart_or_wait",
	)

	return fmt.Errorf("manual recovery requires service restart or waiting for detection cycle")
}

// GetReliabilityStatus returns overall reliability status
func (rm *ReliabilityManager) GetReliabilityStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled": rm.config.Enabled,
	}

	if !rm.config.Enabled {
		return status
	}

	// Add health check status
	if rm.healthChecker != nil {
		healthStatuses := rm.healthChecker.GetAllVMHealth()
		healthyCount := 0
		unhealthyCount := 0

		for _, health := range healthStatuses {
			if health.IsHealthy {
				healthyCount++
			} else {
				unhealthyCount++
			}
		}

		status["health_check"] = map[string]interface{}{
			"total_vms":     len(healthStatuses),
			"healthy_vms":   healthyCount,
			"unhealthy_vms": unhealthyCount,
		}
	}

	// Add recovery status
	if rm.recoveryManager != nil {
		orphanedVMs := rm.recoveryManager.GetOrphanedVMs()

		status["recovery"] = map[string]interface{}{
			"orphaned_vms": len(orphanedVMs),
		}
	}

	// Add registry status
	allVMs := rm.vmRegistry.GetAllVMs()
	status["registry"] = map[string]interface{}{
		"total_vms": len(allVMs),
	}

	return status
}
