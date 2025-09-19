//go:build linux
// +build linux

package firecracker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// CreateVM creates a new VM using the SDK with integrated jailer
func (c *Client) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.create_vm",
		trace.WithAttributes(
			attribute.Int("vcpus", int(config.GetVcpuCount())),
			attribute.Int64("memory_bytes", int64(config.GetMemorySizeMib())),
		),
	)
	defer span.End()

	c.logger.LogAttrs(ctx, slog.LevelInfo, "creating VM",
		slog.String("vm_id", config.GetId()),
		slog.Int("vcpus", int(config.GetVcpuCount())),
		slog.Int64("memory_bytes", int64(config.GetMemorySizeMib())),
	)

	// Create VM directory
	vmDir := filepath.Join(c.baseDir, config.GetId())
	if err := os.MkdirAll(vmDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create VM directory: %w", err)
	}

	c.logger.DebugContext(ctx, "created VM directory",
		slog.String("directory", vmDir),
	)

	// Register the VM
	vm := &VM{
		ID:         config.GetId(),
		Config:     config,
		State:      metaldv1.VmState_VM_STATE_CREATED,
		Machine:    nil, // Will be set when we boot
		CancelFunc: nil, // Will be set when we boot
	}

	c.vmCreateCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm created",
		slog.String("vm_id", vm.ID),
	)

	return vm.ID, nil
}
