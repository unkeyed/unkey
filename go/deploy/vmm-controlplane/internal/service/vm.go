package service

import (
	"context"
	"fmt"
	"log/slog"

	vmmv1 "vmm-controlplane/gen/vmm/v1"
	"vmm-controlplane/gen/vmm/v1/vmmv1connect"
	"vmm-controlplane/internal/backend/types"

	"connectrpc.com/connect"
)

// VMService implements the VmServiceHandler interface
type VMService struct {
	backend types.Backend
	logger  *slog.Logger
	vmmv1connect.UnimplementedVmServiceHandler
}

// NewVMService creates a new VM service instance
func NewVMService(backend types.Backend, logger *slog.Logger) *VMService {
	return &VMService{
		backend: backend,
		logger:  logger.With("service", "vm"),
	}
}

// CreateVm creates a new VM instance
func (s *VMService) CreateVm(ctx context.Context, req *connect.Request[vmmv1.CreateVmRequest]) (*connect.Response[vmmv1.CreateVmResponse], error) {
	s.logger.LogAttrs(ctx, slog.LevelInfo, "creating vm",
		slog.String("method", "CreateVm"),
	)

	config := req.Msg.GetConfig()
	if config == nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm config")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm config is required"))
	}

	// Validate required fields
	if err := s.validateVMConfig(config); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "invalid vm config",
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Create VM using backend (config is already in unified format)
	vmID, err := s.backend.CreateVM(ctx, config)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to create vm",
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm created successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&vmmv1.CreateVmResponse{
		VmId: vmID,
	}), nil
}

// DeleteVm deletes a VM instance
func (s *VMService) DeleteVm(ctx context.Context, req *connect.Request[vmmv1.DeleteVmRequest]) (*connect.Response[vmmv1.DeleteVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "deleting vm",
		slog.String("method", "DeleteVm"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	if err := s.backend.DeleteVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to delete vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm deleted successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&vmmv1.DeleteVmResponse{
		Success: true,
	}), nil
}

// BootVm boots a VM instance
func (s *VMService) BootVm(ctx context.Context, req *connect.Request[vmmv1.BootVmRequest]) (*connect.Response[vmmv1.BootVmResponse], error) {
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

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm booted successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&vmmv1.BootVmResponse{
		Success: true,
	}), nil
}

// ShutdownVm shuts down a VM instance
func (s *VMService) ShutdownVm(ctx context.Context, req *connect.Request[vmmv1.ShutdownVmRequest]) (*connect.Response[vmmv1.ShutdownVmResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down vm",
		slog.String("method", "ShutdownVm"),
		slog.String("vm_id", vmID),
	)

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	if err := s.backend.ShutdownVM(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to shutdown vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to shutdown vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm shutdown successfully",
		slog.String("vm_id", vmID),
	)

	return connect.NewResponse(&vmmv1.ShutdownVmResponse{
		Success: true,
	}), nil
}

// PauseVm pauses a VM instance
func (s *VMService) PauseVm(ctx context.Context, req *connect.Request[vmmv1.PauseVmRequest]) (*connect.Response[vmmv1.PauseVmResponse], error) {
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

	return connect.NewResponse(&vmmv1.PauseVmResponse{
		Success: true,
	}), nil
}

// ResumeVm resumes a paused VM instance
func (s *VMService) ResumeVm(ctx context.Context, req *connect.Request[vmmv1.ResumeVmRequest]) (*connect.Response[vmmv1.ResumeVmResponse], error) {
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

	return connect.NewResponse(&vmmv1.ResumeVmResponse{
		Success: true,
	}), nil
}

// RebootVm reboots a VM instance
func (s *VMService) RebootVm(ctx context.Context, req *connect.Request[vmmv1.RebootVmRequest]) (*connect.Response[vmmv1.RebootVmResponse], error) {
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

	return connect.NewResponse(&vmmv1.RebootVmResponse{
		Success: true,
	}), nil
}

// GetVmInfo gets VM information
func (s *VMService) GetVmInfo(ctx context.Context, req *connect.Request[vmmv1.GetVmInfoRequest]) (*connect.Response[vmmv1.GetVmInfoResponse], error) {
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

	return connect.NewResponse(&vmmv1.GetVmInfoResponse{
		Config: info.Config,
		State:  info.State,
	}), nil
}

// validateVMConfig validates the VM configuration
func (s *VMService) validateVMConfig(config *vmmv1.VmConfig) error {
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

	return nil
}
