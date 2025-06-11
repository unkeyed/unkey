package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/billing"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/observability"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/baggage"
)

// VMService implements the VmServiceHandler interface
type VMService struct {
	backend          types.Backend
	logger           *slog.Logger
	metricsCollector *billing.MetricsCollector
	vmMetrics        *observability.VMMetrics
	vmprovisionerv1connect.UnimplementedVmServiceHandler
}

// NewVMService creates a new VM service instance
func NewVMService(backend types.Backend, logger *slog.Logger, metricsCollector *billing.MetricsCollector, vmMetrics *observability.VMMetrics) *VMService {
	return &VMService{
		backend:          backend,
		logger:           logger.With("service", "vm"),
		metricsCollector: metricsCollector,
		vmMetrics:        vmMetrics,
	}
}

// CreateVm creates a new VM instance
func (s *VMService) CreateVm(ctx context.Context, req *connect.Request[metaldv1.CreateVmRequest]) (*connect.Response[metaldv1.CreateVmResponse], error) {
	s.logger.LogAttrs(ctx, slog.LevelInfo, "creating vm",
		slog.String("method", "CreateVm"),
	)

	// Record VM create request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMCreateRequest(ctx, s.getBackendType())
	}

	config := req.Msg.GetConfig()
	if config == nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm config")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "missing_config")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm config is required"))
	}

	// Validate required fields
	if err := s.validateVMConfig(config); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "invalid vm config",
			slog.String("error", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "invalid_config")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Add tenant context to logs for audit trail
	// AIDEV-NOTE: In multi-tenant systems, all VM operations should be logged with tenant context
	s.logWithTenantContext(ctx, slog.LevelInfo, "creating vm",
		slog.Int("vcpus", int(config.Cpu.VcpuCount)),
		slog.Int64("memory_bytes", config.Memory.SizeBytes),
	)

	// Create VM using backend (config is already in unified format)
	vmID, err := s.backend.CreateVM(ctx, config)
	if err != nil {
		s.logWithTenantContext(ctx, slog.LevelError, "failed to create vm",
			slog.String("error", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "backend_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm created successfully",
		slog.String("vm_id", vmID),
	)

	// Record successful VM creation
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMCreateSuccess(ctx, vmID, s.getBackendType())
	}

	return connect.NewResponse(&metaldv1.CreateVmResponse{
		VmId:  vmID,
		State: metaldv1.VmState_VM_STATE_CREATED,
	}), nil
}

// DeleteVm deletes a VM instance
func (s *VMService) DeleteVm(ctx context.Context, req *connect.Request[metaldv1.DeleteVmRequest]) (*connect.Response[metaldv1.DeleteVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "deleting vm",
		slog.String("method", "DeleteVm"),
		slog.String("vm_id", vmID),
	)

	// Record VM delete request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMDeleteRequest(ctx, vmID, s.getBackendType())
	}

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMDeleteFailure(ctx, "", s.getBackendType(), "missing_vm_id")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	// Stop metrics collection before deletion
	if s.metricsCollector != nil {
		s.metricsCollector.StopCollection(vmID)
		s.logger.LogAttrs(ctx, slog.LevelInfo, "stopped metrics collection",
			slog.String("vm_id", vmID),
		)
	}

	if err := s.backend.DeleteVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to delete vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMDeleteFailure(ctx, vmID, s.getBackendType(), "backend_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm deleted successfully",
		slog.String("vm_id", vmID),
	)

	// Record successful VM deletion
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMDeleteSuccess(ctx, vmID, s.getBackendType())
	}

	return connect.NewResponse(&metaldv1.DeleteVmResponse{
		Success: true,
	}), nil
}

// BootVm boots a VM instance
func (s *VMService) BootVm(ctx context.Context, req *connect.Request[metaldv1.BootVmRequest]) (*connect.Response[metaldv1.BootVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "booting vm",
		slog.String("method", "BootVm"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	if err := s.backend.BootVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to boot vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to boot vm: %w", err))
	}

	// Start metrics collection for billing
	if s.metricsCollector != nil {
		customerID := s.extractCustomerID(ctx, vmID)
		if err := s.metricsCollector.StartCollection(vmID, customerID); err != nil {
			s.logger.LogAttrs(ctx, slog.LevelError, "failed to start metrics collection",
				slog.String("vm_id", vmID),
				slog.String("customer_id", customerID),
				slog.String("error", err.Error()),
			)
			// Don't fail VM boot if metrics collection fails
		} else {
			s.logger.LogAttrs(ctx, slog.LevelInfo, "started metrics collection",
				slog.String("vm_id", vmID),
				slog.String("customer_id", customerID),
			)
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm booted successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&metaldv1.BootVmResponse{
		Success: true,
		State:   metaldv1.VmState_VM_STATE_RUNNING,
	}), nil
}

// ShutdownVm shuts down a VM instance
func (s *VMService) ShutdownVm(ctx context.Context, req *connect.Request[metaldv1.ShutdownVmRequest]) (*connect.Response[metaldv1.ShutdownVmResponse], error) {
	vmID := req.Msg.GetVmId()

	force := req.Msg.GetForce()
	timeout := req.Msg.GetTimeoutSeconds()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down vm",
		slog.String("method", "ShutdownVm"),
		slog.String("vm_id", vmID),
		slog.Bool("force", force),
		slog.Int("timeout_seconds", int(timeout)),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	// Stop metrics collection before shutdown
	if s.metricsCollector != nil {
		s.metricsCollector.StopCollection(vmID)
		s.logger.LogAttrs(ctx, slog.LevelInfo, "stopped metrics collection",
			slog.String("vm_id", vmID),
		)
	}

	if err := s.backend.ShutdownVMWithOptions(ctx, vmID, force, timeout); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to shutdown vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to shutdown vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm shutdown successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&metaldv1.ShutdownVmResponse{
		Success: true,
		State:   metaldv1.VmState_VM_STATE_SHUTDOWN,
	}), nil
}

// PauseVm pauses a VM instance
func (s *VMService) PauseVm(ctx context.Context, req *connect.Request[metaldv1.PauseVmRequest]) (*connect.Response[metaldv1.PauseVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "pausing vm",
		slog.String("method", "PauseVm"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	if err := s.backend.PauseVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to pause vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to pause vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm paused successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&metaldv1.PauseVmResponse{
		Success: true,
		State:   metaldv1.VmState_VM_STATE_PAUSED,
	}), nil
}

// ResumeVm resumes a paused VM instance
func (s *VMService) ResumeVm(ctx context.Context, req *connect.Request[metaldv1.ResumeVmRequest]) (*connect.Response[metaldv1.ResumeVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "resuming vm",
		slog.String("method", "ResumeVm"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	if err := s.backend.ResumeVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to resume vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to resume vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm resumed successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&metaldv1.ResumeVmResponse{
		Success: true,
		State:   metaldv1.VmState_VM_STATE_RUNNING,
	}), nil
}

// RebootVm reboots a VM instance
func (s *VMService) RebootVm(ctx context.Context, req *connect.Request[metaldv1.RebootVmRequest]) (*connect.Response[metaldv1.RebootVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "rebooting vm",
		slog.String("method", "RebootVm"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	if err := s.backend.RebootVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to reboot vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reboot vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm rebooted successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&metaldv1.RebootVmResponse{
		Success: true,
		State:   metaldv1.VmState_VM_STATE_RUNNING,
	}), nil
}

// GetVmInfo gets VM information
func (s *VMService) GetVmInfo(ctx context.Context, req *connect.Request[metaldv1.GetVmInfoRequest]) (*connect.Response[metaldv1.GetVmInfoResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "getting vm info",
		slog.String("method", "GetVmInfo"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	info, err := s.backend.GetVMInfo(ctx, vmID)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to get vm info",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get vm info: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "retrieved vm info successfully",
		slog.String("vm_id", vmID),
		slog.String("state", info.State.String()),
	)

	return connect.NewResponse(&metaldv1.GetVmInfoResponse{
		Config: info.Config,
		State:  info.State,
	}), nil
}

// ListVms lists all VMs managed by this service
func (s *VMService) ListVms(ctx context.Context, req *connect.Request[metaldv1.ListVmsRequest]) (*connect.Response[metaldv1.ListVmsResponse], error) {
	s.logger.LogAttrs(ctx, slog.LevelInfo, "listing vms",
		slog.String("method", "ListVms"),
	)

	var vms []*metaldv1.VmInfo
	var totalCount int32

	if listProvider, ok := s.backend.(types.VMListProvider); ok {
		// Backend supports listing (e.g., ManagedClient)
		managedVMs := listProvider.ListVMs()
		totalCount = int32(len(managedVMs))

		// Convert to protobuf format
		for _, vm := range managedVMs {
			vmInfo := &metaldv1.VmInfo{
				VmId:  vm.ID,
				State: vm.State,
			}

			// Add CPU and memory info if available
			if vm.Config != nil {
				if vm.Config.Cpu != nil {
					vmInfo.VcpuCount = vm.Config.Cpu.VcpuCount
				}
				if vm.Config.Memory != nil {
					vmInfo.MemorySizeBytes = vm.Config.Memory.SizeBytes
				}
				if vm.Config.Metadata != nil {
					vmInfo.Metadata = vm.Config.Metadata
				}
			}

			// Set timestamps (using current time as placeholder)
			now := time.Now().Unix()
			vmInfo.CreatedTimestamp = now
			vmInfo.ModifiedTimestamp = now

			vms = append(vms, vmInfo)
		}
	} else {
		// Backend doesn't support listing (legacy single-VM backends)
		s.logger.LogAttrs(ctx, slog.LevelInfo, "backend does not support vm listing")
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm listing completed",
		slog.Int("count", int(totalCount)),
	)

	return connect.NewResponse(&metaldv1.ListVmsResponse{
		Vms:        vms,
		TotalCount: totalCount,
	}), nil
}

// validateVMConfig validates the VM configuration
func (s *VMService) validateVMConfig(config *metaldv1.VmConfig) error {
	// AIDEV-BUSINESS_RULE: VM configuration must have CPU, memory, and boot settings
	if config.GetCpu() == nil {
		return fmt.Errorf("cpu configuration is required")
	}

	if config.GetMemory() == nil {
		return fmt.Errorf("memory configuration is required")
	}

	if config.GetBoot() == nil {
		return fmt.Errorf("boot configuration is required")
	}

	// Validate CPU configuration
	cpu := config.GetCpu()
	if cpu.GetVcpuCount() <= 0 {
		return fmt.Errorf("vcpu_count must be greater than 0")
	}

	if cpu.GetMaxVcpuCount() > 0 && cpu.GetMaxVcpuCount() < cpu.GetVcpuCount() {
		return fmt.Errorf("max_vcpu_count must be greater than or equal to vcpu_count")
	}

	// Validate memory configuration
	memory := config.GetMemory()
	if memory.GetSizeBytes() <= 0 {
		return fmt.Errorf("memory size_bytes must be greater than 0")
	}

	// Validate boot configuration
	boot := config.GetBoot()
	if boot.GetKernelPath() == "" {
		return fmt.Errorf("kernel_path is required")
	}

	// Validate storage configuration - ensure at least one storage device exists
	if len(config.GetStorage()) == 0 {
		return fmt.Errorf("at least one storage device is required")
	}

	// Validate that we have a root device
	hasRootDevice := false
	for i, storage := range config.GetStorage() {
		if storage.GetPath() == "" {
			return fmt.Errorf("storage device %d path is required", i)
		}
		if storage.GetIsRootDevice() || i == 0 {
			hasRootDevice = true
		}
	}
	if !hasRootDevice {
		return fmt.Errorf("at least one storage device must be marked as root device")
	}

	return nil
}

// extractCustomerID extracts the customer ID for billing from VM context
// This can come from baggage, VM metadata, or other sources
func (s *VMService) extractCustomerID(ctx context.Context, vmID string) string {
	// Try to extract from baggage first (preferred for multi-tenant systems)
	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
		if tenantID := requestBaggage.Member("tenant_id").Value(); tenantID != "" {
			s.logger.LogAttrs(ctx, slog.LevelDebug, "extracted customer ID from baggage",
				slog.String("vm_id", vmID),
				slog.String("customer_id", tenantID),
			)
			return tenantID
		}
	}

	// TODO: Try to extract from VM metadata/config
	// For now, use a placeholder approach
	customerID := "default-customer"
	s.logger.LogAttrs(ctx, slog.LevelWarn, "using default customer ID for billing",
		slog.String("vm_id", vmID),
		slog.String("customer_id", customerID),
	)

	return customerID
}

// logWithTenantContext logs a message with tenant context from baggage for audit trails
// AIDEV-NOTE: Multi-tenant systems require all operations to be logged with tenant context
func (s *VMService) logWithTenantContext(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// Extract tenant context from baggage
	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
		tenantID := requestBaggage.Member("tenant_id").Value()
		userID := requestBaggage.Member("user_id").Value()
		workspaceID := requestBaggage.Member("workspace_id").Value()

		// Add tenant attributes to log
		allAttrs := make([]slog.Attr, 0, len(attrs)+3)
		if tenantID != "" {
			allAttrs = append(allAttrs, slog.String("tenant_id", tenantID))
		}
		if userID != "" {
			allAttrs = append(allAttrs, slog.String("user_id", userID))
		}
		if workspaceID != "" {
			allAttrs = append(allAttrs, slog.String("workspace_id", workspaceID))
		}
		allAttrs = append(allAttrs, attrs...)

		s.logger.LogAttrs(ctx, level, msg, allAttrs...)
	} else {
		// Fallback to regular logging if no baggage
		s.logger.LogAttrs(ctx, level, msg, attrs...)
	}
}

// getBackendType returns the backend type as a string for metrics
func (s *VMService) getBackendType() string {
	// Try to determine backend type from the backend implementation
	switch s.backend.(type) {
	case interface{ GetProcessInfo() map[string]interface{} }:
		return "firecracker"
	default:
		return "cloudhypervisor"
	}
}
