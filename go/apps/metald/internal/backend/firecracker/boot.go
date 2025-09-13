package firecracker

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	sdk "github.com/firecracker-microvm/firecracker-go-sdk"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// BootVM starts a created VM using our integrated jailer
func (c *Client) BootVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.boot_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "boot"),
			attribute.String("error", "vm_not_found"),
		))
		return err
	}

	// Validate VM state before boot operation
	// TODO: This should also boot stopped/paused VMs at some point
	if vm.State != metaldv1.VmState_VM_STATE_CREATED {
		err := fmt.Errorf("vm %s is in %s state, can only boot VMs in CREATED state", vmID, vm.State.String())
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "boot"),
			attribute.String("error", "invalid_state_transition"),
			attribute.String("current_state", vm.State.String()),
		))
		return err
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "booting VM",
		slog.String("vm_id", vmID),
		slog.String("current_state", vm.State.String()),
	)

	// Load VM metadata
	metadata, err := c.prepareVMBootMetadata(ctx, vmID, vm)
	if err != nil {
		c.logger.WarnContext(ctx, "failed to prepare boot metadata",
			"vm_id", vmID,
			"error", err,
		)
		// Continue without metadata rather than failing the boot
	}

	// Build and configure firecracker
	fcConfig := c.configureFirecrackerForBoot(ctx, vmID, vm, metadata)

	// Create a context for this VM
	vmCtx, cancel := context.WithCancel(context.Background())
	vm.CancelFunc = cancel

	// Create and start the machine using SDK
	machine, err := c.startFirecrackerMachine(vmCtx, fcConfig)
	if err != nil {
		cancel()
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "boot"),
			attribute.String("error", "start_machine"),
		))
		return err
	}

	// Update VM state
	vm.Machine = machine
	vm.State = metaldv1.VmState_VM_STATE_RUNNING

	// Acquire asset leases after successful boot
	c.acquireAssetLeases(ctx, vmID, vm.AssetMapping)

	c.vmBootCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM booted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// prepareVMBootMetadata loads container metadata and prepares port mappings for VM boot
func (c *Client) prepareVMBootMetadata(ctx context.Context, vmID string, vm *VM) (*builderv1.ImageMetadata, error) {
	var metadata *builderv1.ImageMetadata

	disk := vm.Config.GetStorage()
	if disk.GetIsRootDevice() {
		// Use chroot path for metadata loading since assets are copied there
		jailerRoot := filepath.Join(c.jailerConfig.ChrootBaseDir, "firecracker", vmID, "root")
		chrootRootfsPath := filepath.Join(jailerRoot, "rootfs.ext4")

		m, metadataErr := c.loadContainerMetadata(ctx, chrootRootfsPath)
		if metadataErr != nil {
			return nil, fmt.Errorf("failed to load container metadata: %w", metadataErr)
		}

		if m != nil {
			metadata = m

			// Create /container.cmd file for metald-init
			if cmdFileErr := c.createContainerCmdFile(ctx, vmID, metadata); cmdFileErr != nil {
				return nil, fmt.Errorf("failed to create container.cmd file: %w", cmdFileErr)
			}

			c.logger.LogAttrs(ctx, slog.LevelInfo, "loaded metadata for VM boot",
				slog.String("vm_id", vmID),
				slog.String("metadata", metadata.String()),
			)
		}
	}

	return metadata, nil
}

// configureFirecrackerForBoot builds and configures firecracker for VM boot
func (c *Client) configureFirecrackerForBoot(ctx context.Context, vmID string, vm *VM, metadata *builderv1.ImageMetadata) sdk.Config {
	vmDir := filepath.Join(c.baseDir, vmID)
	socketPath := filepath.Join(vmDir, "firecracker.sock")

	// Build firecracker config
	fcConfig := c.buildFirecrackerConfig(ctx, vmID, vm.Config, vm.NetworkInfo, vm.AssetPaths)
	fcConfig.SocketPath = socketPath

	// Update kernel args with network configuration and metadata if available
	fcConfig.KernelArgs = c.BuildKernelArgs(ctx, vm.NetworkInfo, metadata)

	// Set the network namespace for the SDK to use
	if vm.NetworkInfo != nil && vm.NetworkInfo.Namespace != "" {
		fcConfig.NetNS = filepath.Join("/run/netns", vm.NetworkInfo.Namespace)
	}

	return fcConfig
}

// startFirecrackerMachine creates and starts the firecracker machine
func (c *Client) startFirecrackerMachine(ctx context.Context, fcConfig sdk.Config) (*sdk.Machine, error) {
	machine, err := sdk.NewMachine(ctx, fcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create firecracker machine: %w", err)
	}

	if err := machine.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start firecracker machine: %w", err)
	}

	return machine, nil
}
