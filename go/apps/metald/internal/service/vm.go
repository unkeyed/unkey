package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/billing"
	"github.com/unkeyed/unkey/go/apps/metald/internal/database"
	"github.com/unkeyed/unkey/go/apps/metald/internal/observability"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// VMService implements the VmServiceHandler interface
type VMService struct {
	backend          types.Backend
	logger           *slog.Logger
	metricsCollector *billing.MetricsCollector
	vmMetrics        *observability.VMMetrics
	tracer           trace.Tracer
	queries          database.Querier
	metaldv1connect.UnimplementedVmServiceHandler
}

// NewVMService creates a new VM service instance
func NewVMService(backend types.Backend, logger *slog.Logger, metricsCollector *billing.MetricsCollector, vmMetrics *observability.VMMetrics, queries database.Querier) *VMService {
	tracer := otel.Tracer("metald.service.vm")
	return &VMService{ //nolint:exhaustruct
		backend:          backend,
		logger:           logger.With("service", "metald"),
		metricsCollector: metricsCollector,
		vmMetrics:        vmMetrics,
		queries:          queries,
		tracer:           tracer,
	}
}

func (s *VMService) generateVmID(ctx context.Context) string {
	b := make([]byte, 15)
	rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)[:24]
}

// CreateVm creates a new VM instance
func (s *VMService) CreateVm(ctx context.Context, req *connect.Request[metaldv1.CreateVmRequest]) (*connect.Response[metaldv1.CreateVmResponse], error) {
	ctx, span := s.tracer.Start(ctx, "metald.vm.create",
		trace.WithAttributes(
			attribute.String("service.name", "metald"),
			attribute.String("operation.name", "create_vm"),
		),
	)
	defer span.End()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "creating vm",
		slog.String("method", "CreateVm"),
	)

	// Record VM create request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMCreateRequest(ctx, s.getBackendType())
	}

	config := req.Msg.GetConfig()

	// DEBUG: Log full request config for debugging
	if config != nil {
		configJSON, _ := json.Marshal(config)
		s.logger.LogAttrs(ctx, slog.LevelDebug, "full VM config received",
			slog.String("config_json", string(configJSON)),
		)
	}
	if config == nil {
		err := fmt.Errorf("vm config is required")
		span.RecordError(err)
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm config")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "missing_config")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	network, netErr := s.queries.AllocateNetwork(ctx)
	if netErr != nil {
		s.logger.Info("failed to allocate network",
			slog.String("error", netErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, netErr)
	}

	s.logger.Info("network allocated",
		slog.Any("network_cidr", network.BaseNetwork),
	)

	ip, ipErr := s.queries.AllocateIP(ctx, database.AllocateIPParams{
		VmID: req.Msg.GetVmId(),
	})
	if ipErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to allocate IP for vm: %w", ipErr))
	}

	// Create VM using backend (config is already in unified format)
	start := time.Now()
	vmID, err := s.backend.CreateVM(ctx, config)
	duration := time.Since(start)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.type", "backend_error"),
			attribute.String("error.message", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "backend_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create vm: %w", err))
	}

	// Record success attributes
	span.SetAttributes(
		attribute.String("vm_id", vmID),
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.Bool("success", true),
	)

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm created successfully",
		slog.String("vm_id", vmID),
		slog.Duration("duration", duration),
	)

	// Record successful VM creation
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMCreateSuccess(ctx, vmID, s.getBackendType(), duration)
	}

	return connect.NewResponse(&metaldv1.CreateVmResponse{
		Endpoint: &metaldv1.Endpoint{
			Host: ip.IpAddr,
			Port: 35428,
		},
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

	// AIDEV-NOTE: Metrics collection re-enabled - metald now reads from Firecracker stats sockets
	// Stop metrics collection before deletion
	if s.metricsCollector != nil {
		s.metricsCollector.StopCollection(vmID)
		s.logger.LogAttrs(ctx, slog.LevelInfo, "stopped metrics collection",
			slog.String("vm_id", vmID),
		)
	}

	start := time.Now()
	err := s.backend.DeleteVM(ctx, vmID)
	duration := time.Since(start)
	if err != nil {
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
		slog.Duration("duration", duration),
	)

	// Record successful VM deletion
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMDeleteSuccess(ctx, vmID, s.getBackendType(), duration)
	}

	return connect.NewResponse(&metaldv1.DeleteVmResponse{
		Success: true,
	}), nil
}

// BootVm boots a VM instance
func (s *VMService) BootVm(ctx context.Context, req *connect.Request[metaldv1.BootVmRequest]) (*connect.Response[metaldv1.BootVmResponse], error) {
	vmID := req.Msg.GetVmId()

	ctx, span := s.tracer.Start(ctx, "metald.vm.boot",
		trace.WithAttributes(
			attribute.String("service.name", "metald"),
			attribute.String("operation.name", "boot_vm"),
			attribute.String("vm_id", vmID),
		),
	)
	defer span.End()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "booting vm",
		slog.String("method", "BootVm"),
		slog.String("vm_id", vmID),
	)

	// Record VM boot request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMBootRequest(ctx, vmID, s.getBackendType())
	}

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMBootFailure(ctx, "", s.getBackendType(), "missing_vm_id")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	start := time.Now()
	err := s.backend.BootVM(ctx, vmID)
	duration := time.Since(start)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.type", "backend_error"),
			attribute.String("error.message", err.Error()),
		)
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to boot vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMBootFailure(ctx, vmID, s.getBackendType(), "backend_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to boot vm: %w", err))
	}

	// Record success attributes
	span.SetAttributes(
		attribute.String("vm_id", vmID),
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.Bool("success", true),
	)

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm booted successfully",
		slog.String("vm_id", vmID),
		slog.Duration("duration", duration),
	)

	// Record successful VM boot
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMBootSuccess(ctx, vmID, s.getBackendType(), duration)
	}

	return connect.NewResponse(&metaldv1.BootVmResponse{
		State: metaldv1.VmState_VM_STATE_RUNNING,
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

	// Record VM shutdown request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMShutdownRequest(ctx, vmID, s.getBackendType(), force)
	}

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMShutdownFailure(ctx, "", s.getBackendType(), force, "missing_vm_id")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	// AIDEV-NOTE: Metrics collection re-enabled - metald now reads from Firecracker stats sockets
	// Stop metrics collection before shutdown
	if s.metricsCollector != nil {
		s.metricsCollector.StopCollection(vmID)
		s.logger.LogAttrs(ctx, slog.LevelInfo, "stopped metrics collection",
			slog.String("vm_id", vmID),
		)
	}

	start := time.Now()
	err := s.backend.ShutdownVMWithOptions(ctx, vmID, force, timeout)
	duration := time.Since(start)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to shutdown vm",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMShutdownFailure(ctx, vmID, s.getBackendType(), force, "backend_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to shutdown vm: %w", err))
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm shutdown successfully",
		slog.String("vm_id", vmID),
		slog.Duration("duration", duration),
	)

	// Record successful VM shutdown
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMShutdownSuccess(ctx, vmID, s.getBackendType(), force, duration)
	}

	return connect.NewResponse(&metaldv1.ShutdownVmResponse{
		State: metaldv1.VmState_VM_STATE_SHUTDOWN,
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
		State: metaldv1.VmState_VM_STATE_PAUSED,
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
		State: metaldv1.VmState_VM_STATE_RUNNING,
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
		State: metaldv1.VmState_VM_STATE_RUNNING,
	}), nil
}

// GetVmInfo gets VM information
func (s *VMService) GetVmInfo(ctx context.Context, req *connect.Request[metaldv1.GetVmInfoRequest]) (*connect.Response[metaldv1.GetVmInfoResponse], error) {
	vmID := req.Msg.GetVmId()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "getting vm info",
		slog.String("method", "GetVmInfo"),
		slog.String("vm_id", vmID),
	)

	// Record VM info request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMInfoRequest(ctx, vmID, s.getBackendType())
	}

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

	return connect.NewResponse(&metaldv1.GetVmInfoResponse{ //nolint:exhaustruct // Metrics and BackendInfo fields are optional and not populated in this response
		VmId:   vmID,
		Config: info.Config,
		State:  info.State,
	}), nil
}

// validateVMConfig validates the VM configuration
// func (s *VMService) validateVMConfig(config *metaldv1.VmConfig) error {
// 	if config.GetCpu() == nil {
// 		return fmt.Errorf("cpu configuration is required")
// 	}

// 	if config.GetMemory() == nil {
// 		return fmt.Errorf("memory configuration is required")
// 	}

// 	if config.GetBoot() == nil {
// 		return fmt.Errorf("boot configuration is required")
// 	}

// 	// Validate CPU configuration
// 	cpu := config.GetCpu()
// 	if cpu.GetVcpuCount() <= 0 {
// 		return fmt.Errorf("vcpu_count must be greater than 0")
// 	}

// 	if cpu.GetMaxVcpuCount() > 0 && cpu.GetMaxVcpuCount() < cpu.GetVcpuCount() {
// 		return fmt.Errorf("max_vcpu_count must be greater than or equal to vcpu_count")
// 	}

// 	// Validate memory configuration
// 	memory := config.GetMemory()
// 	if memory.GetSizeBytes() <= 0 {
// 		return fmt.Errorf("memory size_bytes must be greater than 0")
// 	}

// 	// Validate boot configuration
// 	boot := config.GetBoot()
// 	if boot.GetKernelPath() == "" {
// 		return fmt.Errorf("kernel_path is required")
// 	}

// 	// Validate storage configuration - ensure at least one storage device exists
// 	if len(config.GetStorage()) == 0 {
// 		return fmt.Errorf("at least one storage device is required")
// 	}

// 	// Validate that we have a root device
// 	hasRootDevice := false
// 	for i, storage := range config.GetStorage() {
// 		if storage.GetPath() == "" {
// 			return fmt.Errorf("storage device %d path is required", i)
// 		}
// 		if storage.GetIsRootDevice() || i == 0 {
// 			hasRootDevice = true
// 		}
// 	}
// 	if !hasRootDevice {
// 		return fmt.Errorf("at least one storage device must be marked as root device")
// 	}

// 	return nil
// }

// getBackendType returns the backend type as a string for metrics
func (s *VMService) getBackendType() string {
	return "firecracker"
}
