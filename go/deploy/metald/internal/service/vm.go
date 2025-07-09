package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/billing"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/database"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/observability"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1/vmprovisionerv1connect"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// VMService implements the VmServiceHandler interface
type VMService struct {
	backend          types.Backend
	logger           *slog.Logger
	metricsCollector *billing.MetricsCollector
	vmMetrics        *observability.VMMetrics
	vmRepo           *database.VMRepository
	tracer           trace.Tracer
	vmprovisionerv1connect.UnimplementedVmServiceHandler
}

// NewVMService creates a new VM service instance
func NewVMService(backend types.Backend, logger *slog.Logger, metricsCollector *billing.MetricsCollector, vmMetrics *observability.VMMetrics, vmRepo *database.VMRepository) *VMService {
	tracer := otel.Tracer("metald.service.vm")
	return &VMService{ //nolint:exhaustruct // UnimplementedVmServiceHandler is embedded and provides default implementations
		backend:          backend,
		logger:           logger.With("service", "vm"),
		metricsCollector: metricsCollector,
		vmMetrics:        vmMetrics,
		vmRepo:           vmRepo,
		tracer:           tracer,
	}
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

	// DEBUG: Log full request config for debugging
	config := req.Msg.GetConfig()
	if config != nil {
		configJSON, _ := json.Marshal(config)
		s.logger.LogAttrs(ctx, slog.LevelInfo, "DEBUG: Full VM config received",
			slog.String("config_json", string(configJSON)),
		)
	}

	// Record VM create request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMCreateRequest(ctx, s.getBackendType())
	}

	config := req.Msg.GetConfig()
	if config == nil {
		err := fmt.Errorf("vm config is required")
		span.RecordError(err)
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm config")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "missing_config")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Extract authenticated customer ID from context
	customerID, err := ExtractCustomerID(ctx)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing authenticated customer context")
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "missing_customer_context")
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("customer authentication required"))
	}

	// Validate that request customer_id matches authenticated customer (if provided)
	if req.Msg.GetCustomerId() != "" && req.Msg.GetCustomerId() != customerID {
		s.logger.LogAttrs(ctx, slog.LevelWarn, "SECURITY: customer_id mismatch in request",
			slog.String("authenticated_customer", customerID),
			slog.String("request_customer", req.Msg.GetCustomerId()),
		)
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("customer_id mismatch"))
	}

	// Validate required fields
	if validateErr := s.validateVMConfig(config); validateErr != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "invalid vm config",
			slog.String("error", validateErr.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "invalid_config")
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, validateErr)
	}

	// Add tenant context to logs for audit trail
	// AIDEV-NOTE: In multi-tenant systems, all VM operations should be logged with tenant context
	s.logWithTenantContext(ctx, slog.LevelInfo, "creating vm",
		slog.Int("vcpus", int(config.GetCpu().GetVcpuCount())),
		slog.Int64("memory_bytes", config.GetMemory().GetSizeBytes()),
	)

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
		s.logWithTenantContext(ctx, slog.LevelError, "failed to create vm",
			slog.String("error", err.Error()),
		)
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "backend_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create vm: %w", err))
	}

	// Persist VM to database - critical for state consistency
	if err := s.vmRepo.CreateVMWithContext(ctx, vmID, customerID, config, metaldv1.VmState_VM_STATE_CREATED); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to persist vm to database",
			slog.String("vm_id", vmID),
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()),
		)

		// Attempt robust cleanup with retries to prevent resource leaks
		cleanupSuccess := s.performVMCleanup(ctx, vmID, "database_persistence_failure")
		if !cleanupSuccess {
			// Log critical error - this VM is now orphaned and requires manual intervention
			s.logger.LogAttrs(ctx, slog.LevelError, "CRITICAL: vm cleanup failed after database error - orphaned vm detected",
				slog.String("vm_id", vmID),
				slog.String("customer_id", customerID),
				slog.String("action_required", "manual_cleanup_needed"),
			)
		}

		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMCreateFailure(ctx, s.getBackendType(), "database_error")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to persist vm: %w", err))
	}

	// Record success attributes
	span.SetAttributes(
		attribute.String("vm_id", vmID),
		attribute.String("customer_id", customerID),
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

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMDeleteFailure(ctx, vmID, s.getBackendType(), "ownership_validation_failed")
		}
		return nil, err
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

	// Soft delete VM in database - required for state consistency
	if err := s.vmRepo.DeleteVMWithContext(ctx, vmID); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to delete vm from database",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)

		// Database state consistency is critical - record as partial failure
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMDeleteFailure(ctx, vmID, s.getBackendType(), "database_error")
		}

		// Log warning about state inconsistency but don't fail the operation
		// since backend deletion was successful
		s.logger.LogAttrs(ctx, slog.LevelWarn, "vm delete succeeded in backend but failed in database - state inconsistency detected",
			slog.String("vm_id", vmID),
			slog.String("backend_status", "deleted"),
			slog.String("database_status", "active"),
			slog.String("action_required", "manual_database_cleanup"),
		)
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

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMBootFailure(ctx, vmID, s.getBackendType(), "ownership_validation_failed")
		}
		return nil, err
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

	// Update VM state in database - required for state consistency
	if err := s.vmRepo.UpdateVMStateWithContext(ctx, vmID, metaldv1.VmState_VM_STATE_RUNNING, nil); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to update vm state in database",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)

		// Log warning about state inconsistency
		s.logger.LogAttrs(ctx, slog.LevelWarn, "vm boot succeeded in backend but state update failed in database - state inconsistency detected",
			slog.String("vm_id", vmID),
			slog.String("backend_status", "running"),
			slog.String("database_status", "unknown"),
			slog.String("action_required", "manual_state_sync"),
		)
	}

	// AIDEV-NOTE: Metrics collection re-enabled - metald now reads from Firecracker stats sockets
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

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		if s.vmMetrics != nil {
			s.vmMetrics.RecordVMShutdownFailure(ctx, vmID, s.getBackendType(), force, "ownership_validation_failed")
		}
		return nil, err
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

	// Update VM state in database - required for state consistency
	if err := s.vmRepo.UpdateVMStateWithContext(ctx, vmID, metaldv1.VmState_VM_STATE_SHUTDOWN, nil); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to update vm state in database",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)

		// Log warning about state inconsistency
		s.logger.LogAttrs(ctx, slog.LevelWarn, "vm shutdown succeeded in backend but state update failed in database - state inconsistency detected",
			slog.String("vm_id", vmID),
			slog.String("backend_status", "shutdown"),
			slog.String("database_status", "unknown"),
			slog.String("action_required", "manual_state_sync"),
		)
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

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		return nil, err
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

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		return nil, err
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

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		return nil, err
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

	// Record VM info request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMInfoRequest(ctx, vmID, s.getBackendType())
	}

	if vmID == "" {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing vm id")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vm_id is required"))
	}

	// Validate customer ownership
	if err := s.validateVMOwnership(ctx, vmID); err != nil {
		return nil, err
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
		VmId:        vmID,
		Config:      info.Config,
		State:       info.State,
		NetworkInfo: info.NetworkInfo,
	}), nil
}

// ListVms lists all VMs managed by this service for the authenticated customer
func (s *VMService) ListVms(ctx context.Context, req *connect.Request[metaldv1.ListVmsRequest]) (*connect.Response[metaldv1.ListVmsResponse], error) {
	s.logger.LogAttrs(ctx, slog.LevelInfo, "listing vms",
		slog.String("method", "ListVms"),
	)

	// Record VM list request metric
	if s.vmMetrics != nil {
		s.vmMetrics.RecordVMListRequest(ctx, s.getBackendType())
	}

	// Extract authenticated customer ID for filtering
	customerID, err := ExtractCustomerID(ctx)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "missing authenticated customer context")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("customer authentication required"))
	}

	// Get VMs from database filtered by customer
	dbVMs, err := s.vmRepo.ListVMsByCustomerWithContext(ctx, customerID)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "failed to list vms from database",
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list vms: %w", err))
	}

	var vms []*metaldv1.VmInfo
	// Check for overflow before conversion
	if len(dbVMs) > math.MaxInt32 {
		s.logger.LogAttrs(ctx, slog.LevelError, "too many VMs to list",
			slog.Int("count", len(dbVMs)),
		)
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("too many VMs to list: %d", len(dbVMs)))
	}
	totalCount := int32(len(dbVMs)) //nolint:gosec // Overflow check performed above

	// Convert database VMs to protobuf format
	for _, vm := range dbVMs {
		vmInfo := &metaldv1.VmInfo{ //nolint:exhaustruct // Optional fields are populated conditionally below based on available data
			VmId:       vm.ID,
			State:      vm.State,
			CustomerId: vm.CustomerID,
		}

		// Add CPU and memory info if available
		if vm.ParsedConfig != nil {
			if vm.ParsedConfig.GetCpu() != nil {
				vmInfo.VcpuCount = vm.ParsedConfig.GetCpu().GetVcpuCount()
			}
			if vm.ParsedConfig.GetMemory() != nil {
				vmInfo.MemorySizeBytes = vm.ParsedConfig.GetMemory().GetSizeBytes()
			}
			if vm.ParsedConfig.GetMetadata() != nil {
				vmInfo.Metadata = vm.ParsedConfig.GetMetadata()
			}
		}

		// Set timestamps from database
		vmInfo.CreatedTimestamp = vm.CreatedAt.Unix()
		vmInfo.ModifiedTimestamp = vm.UpdatedAt.Unix()

		vms = append(vms, vmInfo)
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, "vm listing completed",
		slog.Int("count", int(totalCount)),
	)

	return connect.NewResponse(&metaldv1.ListVmsResponse{ //nolint:exhaustruct // NextPageToken field not used as pagination is not implemented yet
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

// extractCustomerID extracts the customer ID for billing from VM database record
// Falls back to baggage context and finally to default customer ID
func (s *VMService) extractCustomerID(ctx context.Context, vmID string) string {
	// First try to get from database (preferred source)
	if vm, err := s.vmRepo.GetVMWithContext(ctx, vmID); err == nil {
		s.logger.LogAttrs(ctx, slog.LevelDebug, "extracted customer ID from database",
			slog.String("vm_id", vmID),
			slog.String("customer_id", vm.CustomerID),
		)
		return vm.CustomerID
	} else {
		s.logger.LogAttrs(ctx, slog.LevelWarn, "failed to get customer ID from database, trying fallback methods",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
	}

	// Fallback to baggage extraction (for compatibility with existing multi-tenant systems)
	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
		if tenantID := requestBaggage.Member("tenant_id").Value(); tenantID != "" {
			s.logger.LogAttrs(ctx, slog.LevelDebug, "extracted customer ID from baggage as fallback",
				slog.String("vm_id", vmID),
				slog.String("customer_id", tenantID),
			)
			return tenantID
		}
	}

	// Final fallback to default customer ID
	customerID := "default-customer"
	s.logger.LogAttrs(ctx, slog.LevelWarn, "using default customer ID for billing",
		slog.String("vm_id", vmID),
		slog.String("customer_id", customerID),
	)

	return customerID
}

// performVMCleanup attempts robust cleanup of a backend VM with retries
// Returns true if cleanup was successful, false if cleanup failed and VM is orphaned
func (s *VMService) performVMCleanup(ctx context.Context, vmID, reason string) bool {
	const maxRetries = 3
	const retryDelay = time.Second
	const cleanupGracePeriod = 30 * time.Second

	// Create a cleanup context with grace period to ensure critical cleanup completes
	// even if the original context is cancelled
	cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupGracePeriod)
	defer cancel()

	s.logger.LogAttrs(ctx, slog.LevelInfo, "attempting vm cleanup",
		slog.String("vm_id", vmID),
		slog.String("reason", reason),
		slog.Int("max_retries", maxRetries),
		slog.Duration("grace_period", cleanupGracePeriod),
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// Wait before retry using cleanup context
			select {
			case <-cleanupCtx.Done():
				s.logger.LogAttrs(ctx, slog.LevelError, "vm cleanup cancelled due to grace period timeout",
					slog.String("vm_id", vmID),
					slog.Int("attempt", attempt),
					slog.Duration("grace_period", cleanupGracePeriod),
				)
				return false
			case <-time.After(retryDelay):
			}
		}

		s.logger.LogAttrs(ctx, slog.LevelDebug, "attempting vm cleanup",
			slog.String("vm_id", vmID),
			slog.Int("attempt", attempt),
		)

		if err := s.backend.DeleteVM(cleanupCtx, vmID); err != nil {
			s.logger.LogAttrs(ctx, slog.LevelWarn, "vm cleanup attempt failed",
				slog.String("vm_id", vmID),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()),
			)

			if attempt == maxRetries {
				s.logger.LogAttrs(ctx, slog.LevelError, "vm cleanup failed after all retries",
					slog.String("vm_id", vmID),
					slog.String("final_error", err.Error()),
				)
				return false
			}
			continue
		}

		s.logger.LogAttrs(ctx, slog.LevelInfo, "vm cleanup successful",
			slog.String("vm_id", vmID),
			slog.Int("attempt", attempt),
		)
		return true
	}

	return false
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
