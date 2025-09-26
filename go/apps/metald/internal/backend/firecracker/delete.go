//go:build linux
// +build linux

package firecracker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// DeleteVM deletes a VM and cleans up all associated resources
func (c *Client) DeleteVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.delete_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	c.logger.LogAttrs(ctx, slog.LevelInfo, "deleting VM",
		slog.String("vm_id", vmID),
	)

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "delete"),
			attribute.String("error", "vm_not_found"),
		))
		return err
	}

	// Stop the VM if it's running
	if vm.Machine != nil {
		if err := vm.Machine.StopVMM(); err != nil {
			c.logger.WarnContext(ctx, "failed to stop VMM during delete",
				"vm_id", vmID,
				"error", err,
			)
		}

		// Cancel the VM context
		if vm.CancelFunc != nil {
			vm.CancelFunc()
		}
	}

	// Clean up VM directory
	vmDir := filepath.Join(c.baseDir, vmID)
	if err := os.RemoveAll(vmDir); err != nil {
		c.logger.WarnContext(ctx, "failed to remove VM directory",
			"vm_id", vmID,
			"path", vmDir,
			"error", err,
		)
	}

	// Clean up jailer chroot
	chrootPath := filepath.Join(c.jailerConfig.ChrootBaseDir, "firecracker", vmID)
	if err := os.RemoveAll(chrootPath); err != nil {
		c.logger.WarnContext(ctx, "failed to remove jailer chroot",
			"vm_id", vmID,
			"path", chrootPath,
			"error", err,
		)
	}

	// Release asset leases
	c.releaseAssetLeases(ctx, vmID)

	// Remove from registry
	delete(c.vmRegistry, vmID)

	c.vmDeleteCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM deleted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}
