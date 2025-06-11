package firecracker

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/process"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ManagedClient implements the Backend interface with automatic Firecracker process management
type ManagedClient struct {
	logger         *slog.Logger
	processManager *process.Manager
	vmRegistry     map[string]*managedVM // vmID -> managedVM
}

type managedVM struct {
	ID        string
	ProcessID string
	Config    *metaldv1.VmConfig
	State     metaldv1.VmState
	Client    *Client // Regular Firecracker client for this specific process
}

// NewManagedClient creates a new managed Firecracker backend client
func NewManagedClient(logger *slog.Logger, appCtx context.Context, pmConfig *config.ProcessManagerConfig) *ManagedClient {
	return &ManagedClient{
		logger:         logger.With("backend", "firecracker-managed"),
		processManager: process.NewManager(logger, appCtx, pmConfig),
		vmRegistry:     make(map[string]*managedVM),
	}
}

// NewManagedClientWithConfig creates a new managed Firecracker backend client with jailer config
func NewManagedClientWithConfig(logger *slog.Logger, appCtx context.Context, pmConfig *config.ProcessManagerConfig, jailerConfig *config.JailerConfig) *ManagedClient {
	return &ManagedClient{
		logger:         logger.With("backend", "firecracker-managed"),
		processManager: process.NewManagerWithConfig(logger, appCtx, pmConfig, jailerConfig),
		vmRegistry:     make(map[string]*managedVM),
	}
}

// Initialize initializes the managed client
func (mc *ManagedClient) Initialize() error {
	mc.logger.Info("initializing managed firecracker client")

	if err := mc.processManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize process manager: %w", err)
	}

	mc.logger.Info("managed firecracker client initialized")
	return nil
}

// CreateVM creates a new VM instance with automatic process management
func (mc *ManagedClient) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	// Generate unique VM ID
	vmID, err := generateVMID()
	if err != nil {
		return "", fmt.Errorf("failed to generate VM ID: %w", err)
	}

	mc.logger.LogAttrs(ctx, slog.LevelInfo, "creating managed vm",
		slog.String("vm_id", vmID),
		slog.Int("vcpus", int(config.Cpu.VcpuCount)),
		slog.Int64("memory_bytes", config.Memory.SizeBytes),
	)

	// Get or create a Firecracker process for this VM
	firecrackerProcess, err := mc.processManager.GetOrCreateProcess(ctx, vmID)
	if err != nil {
		return "", fmt.Errorf("failed to get firecracker process: %w", err)
	}

	// Create a regular Firecracker client that talks to this specific process
	processClient := &Client{
		endpoint:     fmt.Sprintf("unix://%s", firecrackerProcess.SocketPath),
		httpClient:   mc.createHTTPClient(firecrackerProcess.SocketPath),
		logger:       mc.logger,
		metricsFiles: make(map[string]string),
		metricsFIFOs: make(map[string]string),
		collectors:   make(map[string]chan types.VMMetrics),
		lastMetrics:  make(map[string]*types.VMMetrics),
		process:      firecrackerProcess, // Pass the process info
	}

	// Use the regular client to configure the VM with our managed VM ID
	createdVMID, err := processClient.CreateVMWithID(ctx, config, vmID)
	if err != nil {
		// Release the process if VM creation failed
		mc.processManager.ReleaseProcess(ctx, vmID)
		return "", fmt.Errorf("failed to create vm on process: %w", err)
	}

	// Verify that the created VM ID matches our managed VM ID
	if createdVMID != vmID {
		mc.processManager.ReleaseProcess(ctx, vmID)
		return "", fmt.Errorf("VM ID mismatch: expected %s, got %s", vmID, createdVMID)
	}

	// Register the VM
	managedVm := &managedVM{
		ID:        vmID,
		ProcessID: firecrackerProcess.ID,
		Config:    config,
		State:     metaldv1.VmState_VM_STATE_CREATED,
		Client:    processClient,
	}
	mc.vmRegistry[vmID] = managedVm

	mc.logger.LogAttrs(ctx, slog.LevelInfo, "managed vm created successfully",
		slog.String("vm_id", vmID),
		slog.String("process_id", firecrackerProcess.ID),
	)

	return vmID, nil
}

// DeleteVM removes a VM instance and optionally cleans up the process
func (mc *ManagedClient) DeleteVM(ctx context.Context, vmID string) error {
	mc.logger.LogAttrs(ctx, slog.LevelInfo, "deleting managed vm",
		slog.String("vm_id", vmID),
	)

	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	// Delete VM using the specific client
	if err := managedVm.Client.DeleteVM(ctx, vmID); err != nil {
		mc.logger.LogAttrs(ctx, slog.LevelWarn, "failed to delete vm on process",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		// Continue with cleanup even if deletion failed
	}

	// Release the process (makes it available for other VMs)
	if err := mc.processManager.ReleaseProcess(ctx, vmID); err != nil {
		mc.logger.LogAttrs(ctx, slog.LevelWarn, "failed to release process",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
	}

	// Remove from registry
	delete(mc.vmRegistry, vmID)

	mc.logger.LogAttrs(ctx, slog.LevelInfo, "managed vm deleted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// BootVM starts a created VM
func (mc *ManagedClient) BootVM(ctx context.Context, vmID string) error {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	mc.logger.LogAttrs(ctx, slog.LevelInfo, "booting managed vm",
		slog.String("vm_id", vmID),
		slog.String("process_id", managedVm.ProcessID),
	)

	if err := managedVm.Client.BootVM(ctx, vmID); err != nil {
		return err
	}

	managedVm.State = metaldv1.VmState_VM_STATE_RUNNING

	mc.logger.LogAttrs(ctx, slog.LevelInfo, "managed vm booted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ShutdownVM gracefully stops a running VM
func (mc *ManagedClient) ShutdownVM(ctx context.Context, vmID string) error {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	if err := managedVm.Client.ShutdownVM(ctx, vmID); err != nil {
		return err
	}

	managedVm.State = metaldv1.VmState_VM_STATE_SHUTDOWN
	return nil
}

// ShutdownVMWithOptions gracefully stops a running VM with options
func (mc *ManagedClient) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	if err := managedVm.Client.ShutdownVMWithOptions(ctx, vmID, force, timeoutSeconds); err != nil {
		return err
	}

	managedVm.State = metaldv1.VmState_VM_STATE_SHUTDOWN
	return nil
}

// PauseVM pauses a running VM
func (mc *ManagedClient) PauseVM(ctx context.Context, vmID string) error {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	if err := managedVm.Client.PauseVM(ctx, vmID); err != nil {
		return err
	}

	managedVm.State = metaldv1.VmState_VM_STATE_PAUSED
	return nil
}

// ResumeVM resumes a paused VM
func (mc *ManagedClient) ResumeVM(ctx context.Context, vmID string) error {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	if err := managedVm.Client.ResumeVM(ctx, vmID); err != nil {
		return err
	}

	managedVm.State = metaldv1.VmState_VM_STATE_RUNNING
	return nil
}

// RebootVM restarts a running VM
func (mc *ManagedClient) RebootVM(ctx context.Context, vmID string) error {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return fmt.Errorf("vm %s not found", vmID)
	}

	if err := managedVm.Client.RebootVM(ctx, vmID); err != nil {
		return err
	}

	managedVm.State = metaldv1.VmState_VM_STATE_RUNNING
	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (mc *ManagedClient) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	managedVm, exists := mc.vmRegistry[vmID]
	if !exists {
		return nil, fmt.Errorf("vm %s not found", vmID)
	}

	mc.logger.LogAttrs(ctx, slog.LevelInfo, "getting managed vm info",
		slog.String("vm_id", vmID),
		slog.String("process_id", managedVm.ProcessID),
	)

	return &types.VMInfo{
		Config: managedVm.Config,
		State:  managedVm.State,
	}, nil
}

// Ping checks if the backend is healthy and responsive
func (mc *ManagedClient) Ping(ctx context.Context) error {
	mc.logger.LogAttrs(ctx, slog.LevelDebug, "pinging managed firecracker backend")

	// Check if process manager is healthy
	processInfo := mc.processManager.GetProcessInfo()
	mc.logger.LogAttrs(ctx, slog.LevelDebug, "process manager status",
		slog.Int("active_processes", len(processInfo)),
		slog.Int("managed_vms", len(mc.vmRegistry)),
	)

	mc.logger.LogAttrs(ctx, slog.LevelDebug, "managed firecracker ping successful")
	return nil
}

// Shutdown gracefully shuts down all VMs and processes
func (mc *ManagedClient) Shutdown(ctx context.Context) error {
	mc.logger.Info("shutting down managed firecracker client")

	// Shutdown all VMs first
	for vmID := range mc.vmRegistry {
		if err := mc.DeleteVM(ctx, vmID); err != nil {
			mc.logger.LogAttrs(ctx, slog.LevelError, "failed to delete vm during shutdown",
				slog.String("vm_id", vmID),
				slog.String("error", err.Error()),
			)
		}
	}

	// Shutdown process manager
	if err := mc.processManager.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown process manager: %w", err)
	}

	mc.logger.Info("managed firecracker client shutdown complete")
	return nil
}

// ListVMs returns all managed VMs
func (mc *ManagedClient) ListVMs() []types.ListableVMInfo {
	vms := make([]types.ListableVMInfo, 0, len(mc.vmRegistry))
	for _, vm := range mc.vmRegistry {
		vms = append(vms, types.ListableVMInfo{
			ID:     vm.ID,
			State:  vm.State,
			Config: vm.Config,
		})
	}
	return vms
}

// GetProcessInfo returns information about managed processes
func (mc *ManagedClient) GetProcessInfo() map[string]*process.FirecrackerProcess {
	return mc.processManager.GetProcessInfo()
}

// createHTTPClient creates an HTTP client for the specific socket
func (mc *ManagedClient) createHTTPClient(socketPath string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}

	instrumentedTransport := otelhttp.NewTransport(transport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("Firecracker %s %s", r.Method, r.URL.Path)
		}),
	)

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: instrumentedTransport,
	}
}

// GetVMMetrics retrieves current VM resource usage metrics from managed Firecracker instance
func (mc *ManagedClient) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	managedVM, exists := mc.vmRegistry[vmID]
	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	if managedVM.Client == nil {
		return nil, fmt.Errorf("no client available for VM %s", vmID)
	}

	// Delegate to the underlying Firecracker client
	return managedVM.Client.GetVMMetrics(ctx, vmID)
}

// Ensure ManagedClient implements Backend interface
var _ types.Backend = (*ManagedClient)(nil)
