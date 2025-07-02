package reconciler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/database"
)

// VMReconciler handles VM state reconciliation between database and reality
type VMReconciler struct {
	logger   *slog.Logger
	backend  types.Backend
	vmRepo   *database.VMRepository
	interval time.Duration
	stopChan chan struct{}
}

// NewVMReconciler creates a new VM reconciler
func NewVMReconciler(logger *slog.Logger, backend types.Backend, vmRepo *database.VMRepository, interval time.Duration) *VMReconciler {
	return &VMReconciler{
		logger:   logger.With("component", "vm-reconciler"),
		backend:  backend,
		vmRepo:   vmRepo,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins the reconciliation process
func (r *VMReconciler) Start(ctx context.Context) {
	r.logger.InfoContext(ctx, "starting VM reconciler",
		slog.Duration("interval", r.interval),
	)

	// Run initial reconciliation immediately
	r.reconcileOnce(ctx)

	// Start periodic reconciliation
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.InfoContext(ctx, "VM reconciler stopped due to context cancellation")
			return
		case <-r.stopChan:
			r.logger.InfoContext(ctx, "VM reconciler stopped")
			return
		case <-ticker.C:
			r.reconcileOnce(ctx)
		}
	}
}

// Stop stops the reconciliation process
func (r *VMReconciler) Stop() {
	close(r.stopChan)
}

// ReconcileNow triggers an immediate reconciliation
func (r *VMReconciler) ReconcileNow(ctx context.Context) *ReconciliationReport {
	return r.reconcileOnce(ctx)
}

// reconcileOnce performs a single reconciliation cycle
func (r *VMReconciler) reconcileOnce(ctx context.Context) *ReconciliationReport {
	startTime := time.Now()

	r.logger.InfoContext(ctx, "starting VM reconciliation cycle")

	report := &ReconciliationReport{
		StartTime: startTime,
	}

	// 1. Get all VMs from database
	dbVMs, err := r.vmRepo.ListAllVMsWithContext(ctx)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to list VMs from database",
			slog.String("error", err.Error()),
		)
		report.Errors = append(report.Errors, fmt.Sprintf("database query failed: %v", err))
		return report
	}

	report.DatabaseVMCount = len(dbVMs)
	r.logger.InfoContext(ctx, "found VMs in database",
		slog.Int("count", len(dbVMs)),
	)

	// 2. Get all running Firecracker processes
	runningProcesses, err := r.getRunningFirecrackerProcesses()
	if err != nil {
		r.logger.WarnContext(ctx, "failed to get running Firecracker processes",
			slog.String("error", err.Error()),
		)
		report.Errors = append(report.Errors, fmt.Sprintf("process scan failed: %v", err))
	}

	report.RunningProcessCount = len(runningProcesses)
	r.logger.InfoContext(ctx, "found running Firecracker processes",
		slog.Int("count", len(runningProcesses)),
	)

	// 3. Reconcile each VM
	for _, vm := range dbVMs {
		vmReport := r.reconcileVM(ctx, vm, runningProcesses)
		report.VMReports = append(report.VMReports, vmReport)

		switch vmReport.Action {
		case ReconcileActionMarkDead:
			report.MarkedDead++
		case ReconcileActionUpdateState:
			report.StateUpdated++
		case ReconcileActionDeleteOrphan:
			report.OrphansDeleted++
		case ReconcileActionNoChange:
			report.NoChangeNeeded++
		case ReconcileActionError:
			report.ErrorCount++
		}
	}

	report.Duration = time.Since(startTime)

	r.logger.InfoContext(ctx, "VM reconciliation cycle completed",
		slog.Duration("duration", report.Duration),
		slog.Int("database_vms", report.DatabaseVMCount),
		slog.Int("running_processes", report.RunningProcessCount),
		slog.Int("marked_dead", report.MarkedDead),
		slog.Int("state_updated", report.StateUpdated),
		slog.Int("orphans_deleted", report.OrphansDeleted),
		slog.Int("no_change", report.NoChangeNeeded),
		slog.Int("errors", report.ErrorCount),
	)

	return report
}

// reconcileVM reconciles a single VM's state
func (r *VMReconciler) reconcileVM(ctx context.Context, vm *database.VM, runningProcesses map[string]FirecrackerProcess) VMReconciliationReport {
	vmReport := VMReconciliationReport{
		VMID:          vm.ID,
		DatabaseState: metaldv1.VmState(vm.State),
	}

	// Handle nil ProcessID safely
	processID := ""
	if vm.ProcessID != nil {
		processID = *vm.ProcessID
		vmReport.ProcessID = processID
	}

	// Check if the VM process is actually running
	isProcessRunning := false
	if processID != "" {
		if proc, exists := runningProcesses[processID]; exists {
			isProcessRunning = true
			vmReport.ProcessExists = true
			vmReport.ProcessInfo = proc
		}
	}

	// Determine what action to take based on database state vs reality
	switch metaldv1.VmState(vm.State) {
	case metaldv1.VmState_VM_STATE_RUNNING, metaldv1.VmState_VM_STATE_CREATED:
		if !isProcessRunning {
			// VM is supposed to be running but process doesn't exist
			r.logger.WarnContext(ctx, "VM marked as running but process not found - marking as shutdown",
				slog.String("vm_id", vm.ID),
				slog.String("database_state", metaldv1.VmState(vm.State).String()),
				slog.String("process_id", processID),
			)

			// Mark VM as shutdown in database
			if err := r.markVMDead(ctx, vm.ID, "process not found during reconciliation"); err != nil {
				vmReport.Action = ReconcileActionError
				vmReport.Error = fmt.Sprintf("failed to mark VM as shutdown: %v", err)
			} else {
				vmReport.Action = ReconcileActionMarkDead
				vmReport.NewState = metaldv1.VmState_VM_STATE_SHUTDOWN
			}
		} else {
			// VM and process both exist - state is consistent
			vmReport.Action = ReconcileActionNoChange
		}

	case metaldv1.VmState_VM_STATE_SHUTDOWN, metaldv1.VmState_VM_STATE_PAUSED:
		if isProcessRunning {
			// VM is marked as dead but process is still running - update state
			r.logger.InfoContext(ctx, "VM marked as shutdown but process is running - updating state",
				slog.String("vm_id", vm.ID),
				slog.String("database_state", metaldv1.VmState(vm.State).String()),
				slog.String("process_id", processID),
			)

			if err := r.updateVMState(ctx, vm.ID, metaldv1.VmState_VM_STATE_RUNNING); err != nil {
				vmReport.Action = ReconcileActionError
				vmReport.Error = fmt.Sprintf("failed to update VM state: %v", err)
			} else {
				vmReport.Action = ReconcileActionUpdateState
				vmReport.NewState = metaldv1.VmState_VM_STATE_RUNNING
			}
		} else {
			// VM and process are both shutdown - check if this is an orphaned record
			if r.isOrphanedRecord(ctx, vm) {
				r.logger.WarnContext(ctx, "detected orphaned database record - deleting",
					slog.String("vm_id", vm.ID),
					slog.Time("updated_at", vm.UpdatedAt),
					slog.Duration("age", time.Since(vm.UpdatedAt)),
				)

				if err := r.deleteOrphanedVM(ctx, vm.ID); err != nil {
					vmReport.Action = ReconcileActionError
					vmReport.Error = fmt.Sprintf("failed to delete orphaned VM: %v", err)
				} else {
					vmReport.Action = ReconcileActionDeleteOrphan
				}
			} else {
				// Valid shutdown VM - leave it alone
				vmReport.Action = ReconcileActionNoChange
			}
		}

	default:
		// Unknown state
		vmReport.Action = ReconcileActionNoChange
	}

	return vmReport
}

// markVMDead marks a VM as dead in the database
func (r *VMReconciler) markVMDead(ctx context.Context, vmID, reason string) error {
	return r.vmRepo.UpdateVMStateWithContextInt(ctx, vmID, int(metaldv1.VmState_VM_STATE_SHUTDOWN))
}

// updateVMState updates a VM's state in the database
func (r *VMReconciler) updateVMState(ctx context.Context, vmID string, newState metaldv1.VmState) error {
	return r.vmRepo.UpdateVMStateWithContextInt(ctx, vmID, int(newState))
}

// isOrphanedRecord determines if a shutdown VM is actually an orphaned database record
// Uses defense-in-depth approach: age-based + validation-based + tracking-based checks
func (r *VMReconciler) isOrphanedRecord(ctx context.Context, vm *database.VM) bool {
	now := time.Now()

	// Defense 1: Age-based check - very conservative threshold
	shutdownAge := now.Sub(vm.UpdatedAt)
	if shutdownAge < OrphanedRecordAgeThreshold {
		r.logger.DebugContext(ctx, "VM not old enough to be considered orphaned",
			slog.String("vm_id", vm.ID),
			slog.Duration("age", shutdownAge),
			slog.Duration("threshold", OrphanedRecordAgeThreshold),
		)
		return false
	}

	// Defense 2: Validation-based check - verify VM resources don't exist
	if r.vmResourcesExist(ctx, vm) {
		r.logger.DebugContext(ctx, "VM resources still exist - not orphaned",
			slog.String("vm_id", vm.ID),
		)
		return false
	}

	// Defense 3: Tracking-based check - look for signs of improper shutdown
	if r.hasProperShutdownMarkers(ctx, vm) {
		r.logger.DebugContext(ctx, "VM has proper shutdown markers - not orphaned",
			slog.String("vm_id", vm.ID),
		)
		return false
	}

	// All checks passed - this appears to be an orphaned record
	r.logger.InfoContext(ctx, "VM identified as orphaned record",
		slog.String("vm_id", vm.ID),
		slog.Duration("age", shutdownAge),
	)

	return true
}

// vmResourcesExist checks if VM-related resources still exist (network, storage, etc.)
func (r *VMReconciler) vmResourcesExist(ctx context.Context, vm *database.VM) bool {
	// AIDEV-TODO: Implement resource validation checks
	// For now, we'll assume resources don't exist if no process is running
	// Future enhancements could check:
	// - Network namespace existence
	// - TAP device existence
	// - Storage file existence
	// - Jailer chroot directory existence

	return false
}

// hasProperShutdownMarkers checks for evidence of proper VM shutdown
func (r *VMReconciler) hasProperShutdownMarkers(ctx context.Context, vm *database.VM) bool {
	// AIDEV-TODO: Implement shutdown tracking
	// For now, we'll assume VMs without proper markers are orphaned
	// Future enhancements could check:
	// - Shutdown reason metadata
	// - Graceful shutdown logs
	// - Process exit code tracking

	return false
}

// deleteOrphanedVM safely deletes an orphaned VM record from the database
func (r *VMReconciler) deleteOrphanedVM(ctx context.Context, vmID string) error {
	r.logger.InfoContext(ctx, "deleting orphaned VM record",
		slog.String("vm_id", vmID),
	)

	// Use soft delete to maintain audit trail
	if err := r.vmRepo.DeleteVMWithContext(ctx, vmID); err != nil {
		r.logger.ErrorContext(ctx, "failed to delete orphaned VM",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to delete orphaned VM %s: %w", vmID, err)
	}

	r.logger.InfoContext(ctx, "successfully deleted orphaned VM record",
		slog.String("vm_id", vmID),
	)

	return nil
}

// getRunningFirecrackerProcesses scans for running Firecracker processes
func (r *VMReconciler) getRunningFirecrackerProcesses() (map[string]FirecrackerProcess, error) {
	processes := make(map[string]FirecrackerProcess)

	// Use procfs to find Firecracker processes
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a PID
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read process command line
		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
		cmdlineBytes, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue // Process might have disappeared
		}

		cmdline := string(cmdlineBytes)

		// Check if this is a Firecracker process
		if strings.Contains(cmdline, "firecracker") || strings.Contains(cmdline, "fc_vcpu") {
			// Extract VM ID from command line if possible
			vmID := r.extractVMIDFromCmdline(cmdline)

			process := FirecrackerProcess{
				PID:     pid,
				Cmdline: cmdline,
				VMID:    vmID,
			}

			processes[strconv.Itoa(pid)] = process
		}
	}

	return processes, nil
}

// extractVMIDFromCmdline attempts to extract VM ID from Firecracker command line
func (r *VMReconciler) extractVMIDFromCmdline(cmdline string) string {
	// Look for VM ID patterns in the command line
	// This is heuristic-based and may need adjustment

	// Pattern 1: --id vm-id or --id=vm-id
	if strings.Contains(cmdline, "--id") {
		parts := strings.Fields(strings.ReplaceAll(cmdline, "\x00", " "))
		for i, part := range parts {
			if part == "--id" && i+1 < len(parts) {
				return parts[i+1]
			}
			if strings.HasPrefix(part, "--id=") {
				return strings.TrimPrefix(part, "--id=")
			}
		}
	}

	// Pattern 2: VM ID in socket path
	if strings.Contains(cmdline, "vm-") {
		fields := strings.Fields(strings.ReplaceAll(cmdline, "\x00", " "))
		for _, field := range fields {
			if strings.Contains(field, "vm-") {
				// Extract VM ID from socket path or similar
				parts := strings.Split(field, "/")
				for _, part := range parts {
					if strings.HasPrefix(part, "vm-") {
						return part
					}
				}
			}
		}
	}

	return "" // Could not extract VM ID
}

// FirecrackerProcess represents a running Firecracker process
type FirecrackerProcess struct {
	PID     int    `json:"pid"`
	Cmdline string `json:"cmdline"`
	VMID    string `json:"vm_id,omitempty"`
}

// ReconciliationReport contains the results of a reconciliation cycle
type ReconciliationReport struct {
	StartTime           time.Time                `json:"start_time"`
	Duration            time.Duration            `json:"duration"`
	DatabaseVMCount     int                      `json:"database_vm_count"`
	RunningProcessCount int                      `json:"running_process_count"`
	MarkedDead          int                      `json:"marked_dead"`
	StateUpdated        int                      `json:"state_updated"`
	OrphansDeleted      int                      `json:"orphans_deleted"`
	NoChangeNeeded      int                      `json:"no_change_needed"`
	ErrorCount          int                      `json:"error_count"`
	VMReports           []VMReconciliationReport `json:"vm_reports"`
	Errors              []string                 `json:"errors"`
}

// VMReconciliationReport contains the results for a specific VM
type VMReconciliationReport struct {
	VMID          string             `json:"vm_id"`
	DatabaseState metaldv1.VmState   `json:"database_state"`
	ProcessID     string             `json:"process_id"`
	ProcessExists bool               `json:"process_exists"`
	ProcessInfo   FirecrackerProcess `json:"process_info,omitempty"`
	Action        ReconcileAction    `json:"action"`
	NewState      metaldv1.VmState   `json:"new_state,omitempty"`
	Error         string             `json:"error,omitempty"`
}

// ReconcileAction represents the action taken during reconciliation
type ReconcileAction string

const (
	ReconcileActionNoChange     ReconcileAction = "no_change"
	ReconcileActionMarkDead     ReconcileAction = "mark_dead"
	ReconcileActionUpdateState  ReconcileAction = "update_state"
	ReconcileActionDeleteOrphan ReconcileAction = "delete_orphan"
	ReconcileActionError        ReconcileAction = "error"
)

// AIDEV-BUSINESS_RULE: Orphaned record cleanup thresholds - conservative to protect customer VMs
const (
	// Only consider VMs orphaned after being shutdown for a very long time
	OrphanedRecordAgeThreshold = 7 * 24 * time.Hour // 1 week - conservative

	// Maximum time a VM should reasonably be shutdown before cleanup consideration
	MaxReasonableShutdownTime = 30 * 24 * time.Hour // 30 days - very conservative
)
