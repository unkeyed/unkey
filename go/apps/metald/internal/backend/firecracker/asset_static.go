//go:build linux
// +build linux

package firecracker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// copyMetadataForRootDevice copies metadata and creates container.cmd for a root device
func (c *Client) copyMetadataForRootDevice(ctx context.Context, vmID string, disk *metaldv1.StorageDevice, jailerRoot string, diskDst string) error {
	baseName := strings.TrimSuffix(filepath.Base(disk.GetPath()), filepath.Ext(disk.GetPath()))
	metadataSrc := filepath.Join(filepath.Dir(disk.GetPath()), baseName+".metadata.json")

	// Check if metadata file exists
	if _, err := os.Stat(metadataSrc); err != nil {
		if os.IsNotExist(err) {
			return nil // No metadata file is OK
		}
		return fmt.Errorf("failed to stat metadata file: %w", err)
	}

	// Copy metadata file
	metadataDst := filepath.Join(jailerRoot, filepath.Base(metadataSrc))
	if err := copyFileWithOwnership(metadataSrc, metadataDst, int(c.jailerConfig.UID), int(c.jailerConfig.GID)); err != nil {
		return fmt.Errorf("failed to copy metadata file: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "copied metadata file to jailer root",
		slog.String("src", metadataSrc),
		slog.String("dst", metadataDst),
	)

	// Load and process metadata to create container.cmd
	metadata, err := c.loadContainerMetadata(ctx, disk.GetPath())
	if err != nil || metadata == nil {
		return nil // Can't create container.cmd without metadata
	}

	// Build the command array
	var fullCmd []string
	fullCmd = append(fullCmd, metadata.GetEntrypoint()...)
	fullCmd = append(fullCmd, metadata.GetCommand()...)

	if len(fullCmd) == 0 {
		return nil // No command to write
	}

	// Write command file to rootfs by mounting it temporarily
	return c.writeContainerCmdToRootfs(ctx, vmID, diskDst, fullCmd)
}

// writeContainerCmdToRootfs mounts the rootfs and writes the container.cmd file
func (c *Client) writeContainerCmdToRootfs(ctx context.Context, vmID string, diskDst string, fullCmd []string) error {
	// Create temporary mount directory
	mountDir := filepath.Join("/tmp", fmt.Sprintf("mount-%s", vmID))
	if err := os.MkdirAll(mountDir, 0o755); err != nil {
		return fmt.Errorf("failed to create mount directory: %w", err)
	}
	defer os.RemoveAll(mountDir)

	// Mount the rootfs ext4 image
	mountCmd := exec.CommandContext(ctx, "mount", "-o", "loop", diskDst, mountDir)
	if err := mountCmd.Run(); err != nil {
		return fmt.Errorf("failed to mount rootfs: %w", err)
	}
	defer func() {
		// Always unmount
		umountCmd := exec.CommandContext(ctx, "umount", mountDir)
		if err := umountCmd.Run(); err != nil {
			c.logger.WarnContext(ctx, "failed to unmount rootfs",
				"error", err,
				"mountDir", mountDir,
			)
		}
	}()

	// Write the command file
	cmdFile := filepath.Join(mountDir, "container.cmd")
	cmdData, err := json.Marshal(fullCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := os.WriteFile(cmdFile, cmdData, 0o600); err != nil {
		return fmt.Errorf("failed to write command file: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "wrote container command file to rootfs",
		slog.String("path", cmdFile),
		slog.String("command", string(cmdData)),
	)

	return nil
}
