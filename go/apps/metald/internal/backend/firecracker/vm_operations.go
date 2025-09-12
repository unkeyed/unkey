package firecracker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ShutdownVMWithOptions shuts down a VM with configurable options
func (c *Client) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.shutdown_vm",
		trace.WithAttributes(
			attribute.String("vm_id", vmID),
			attribute.Bool("force", force),
			attribute.Int("timeout_seconds", int(timeoutSeconds)),
		),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return err
	}

	// Validate VM state before shutdown operation
	if vm.State != metaldv1.VmState_VM_STATE_RUNNING {
		err := fmt.Errorf("vm %s is in %s state, can only shutdown VMs in RUNNING state", vmID, vm.State.String())
		span.RecordError(err)
		return err
	}

	if vm.Machine == nil {
		return fmt.Errorf("vm %s firecracker process not available", vmID)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down VM",
		slog.String("vm_id", vmID),
		slog.String("current_state", vm.State.String()),
		slog.Bool("force", force),
		slog.Int("timeout_seconds", int(timeoutSeconds)),
	)

	// Create a timeout context
	shutdownCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if force {
		// Force shutdown by pausing the VM to preserve the socket for resume
		// Note: Using PauseVM instead of StopVMM to allow resume operations
		if err := vm.Machine.PauseVM(shutdownCtx); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to force shutdown VM: %w", err)
		}
	} else {
		// Try graceful shutdown first by pausing the VM
		// Note: Using PauseVM instead of Shutdown to preserve the firecracker process and socket
		if err := vm.Machine.PauseVM(shutdownCtx); err != nil {
			c.logger.WarnContext(ctx, "graceful shutdown failed",
				"vm_id", vmID,
				"error", err,
			)
			span.RecordError(err)
			return fmt.Errorf("failed to shutdown VM: %w", err)
		}
	}

	// Note: The firecracker process remains running to allow resume operations

	// Update state
	vm.State = metaldv1.VmState_VM_STATE_SHUTDOWN

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM shutdown successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// PauseVM pauses a running VM
func (c *Client) PauseVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.pause_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return err
	}

	// Validate VM state before pause operation
	if vm.State != metaldv1.VmState_VM_STATE_RUNNING {
		err := fmt.Errorf("vm %s is in %s state, can only pause VMs in RUNNING state", vmID, vm.State.String())
		span.RecordError(err)
		return err
	}

	if vm.Machine == nil {
		return fmt.Errorf("vm %s firecracker process not available", vmID)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "pausing VM",
		slog.String("vm_id", vmID),
		slog.String("current_state", vm.State.String()),
	)

	if err := vm.Machine.PauseVM(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to pause VM: %w", err)
	}

	vm.State = metaldv1.VmState_VM_STATE_PAUSED

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM paused successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ResumeVM resumes a paused or shutdown VM
func (c *Client) ResumeVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.resume_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return err
	}

	// Validate VM state before resume operation - allow both PAUSED and SHUTDOWN
	if vm.State != metaldv1.VmState_VM_STATE_PAUSED && vm.State != metaldv1.VmState_VM_STATE_SHUTDOWN {
		err := fmt.Errorf("vm %s is in %s state, can only resume VMs in PAUSED or SHUTDOWN state", vmID, vm.State.String())
		span.RecordError(err)
		return err
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "resuming VM",
		slog.String("vm_id", vmID),
		slog.String("current_state", vm.State.String()),
	)

	if err := vm.Machine.ResumeVM(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to resume VM: %w", err)
	}

	vm.State = metaldv1.VmState_VM_STATE_RUNNING

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM resumed successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// RebootVM reboots a running VM
func (c *Client) RebootVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.reboot_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	c.logger.LogAttrs(ctx, slog.LevelInfo, "rebooting VM",
		slog.String("vm_id", vmID),
	)

	// Shutdown the VM
	if err := c.ShutdownVMWithOptions(ctx, vmID, false, 30); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to shutdown VM for reboot: %w", err)
	}

	// Wait a moment
	time.Sleep(1 * time.Second)

	// Resume the VM (since we paused it in shutdown)
	if err := c.ResumeVM(ctx, vmID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to resume VM after shutdown: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM rebooted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// GetVMInfo returns information about a VM
func (c *Client) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	_, span := c.tracer.Start(ctx, "metald.firecracker.get_vm_info",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return nil, err
	}

	info := &types.VMInfo{
		Config: vm.Config,
		State:  vm.State,
	}

	return info, nil
}
