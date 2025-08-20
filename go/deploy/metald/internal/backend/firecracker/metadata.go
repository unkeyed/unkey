package firecracker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// loadContainerMetadata loads container metadata from the metadata file if it exists
func (c *Client) loadContainerMetadata(ctx context.Context, rootfsPath string) (*builderv1.ImageMetadata, error) {
	// Load container metadata saved by builderd
	// The metadata file is named {buildID}.metadata.json and should be alongside the rootfs

	// Extract base name without extension
	baseName := strings.TrimSuffix(filepath.Base(rootfsPath), filepath.Ext(rootfsPath))
	metadataPath := filepath.Join(filepath.Dir(rootfsPath), baseName+".metadata.json")

	c.logger.LogAttrs(ctx, slog.LevelInfo, "looking for container metadata",
		slog.String("rootfs_path", rootfsPath),
		slog.String("metadata_path", metadataPath),
	)

	// Check if metadata file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		// Fallback to check for metadata.json in VM chroot directory
		// When assets are copied to VM chroot by assetmanagerd, metadata file is renamed to metadata.json
		fallbackPath := filepath.Join(filepath.Dir(rootfsPath), "metadata.json")
		if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
			c.logger.LogAttrs(ctx, slog.LevelDebug, "no metadata file found in either location",
				slog.String("primary_path", metadataPath),
				slog.String("fallback_path", fallbackPath),
			)
			return nil, nil // No metadata is not an error
		}
		// Use fallback path
		metadataPath = fallbackPath
		c.logger.LogAttrs(ctx, slog.LevelInfo, "using fallback metadata path",
			slog.String("fallback_path", fallbackPath),
		)
	}

	// Read metadata file
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// Parse metadata
	var metadata builderv1.ImageMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "loaded container metadata",
		slog.String("image", metadata.GetOriginalImage()),
		slog.Int("entrypoint_len", len(metadata.GetEntrypoint())),
		slog.Int("cmd_len", len(metadata.GetCommand())),
		slog.Int("env_vars", len(metadata.GetEnv())),
		slog.Int("exposed_ports", len(metadata.GetExposedPorts())),
	)

	return &metadata, nil
}

// createContainerCmdFile creates /container.cmd file in VM chroot for metald-init
func (c *Client) createContainerCmdFile(ctx context.Context, vmID string, metadata *builderv1.ImageMetadata) error {
	// Create container.cmd file containing the full command for metald-init
	// Combines entrypoint and command from container metadata into JSON array

	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	// Build full command array: entrypoint + command
	var fullCmd []string
	fullCmd = append(fullCmd, metadata.GetEntrypoint()...)
	fullCmd = append(fullCmd, metadata.GetCommand()...)

	if len(fullCmd) == 0 {
		return fmt.Errorf("no entrypoint or command found in metadata")
	}

	// Convert to JSON
	cmdJSON, err := json.Marshal(fullCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command to JSON: %w", err)
	}

	// Write container.cmd into the rootfs.ext4 filesystem, not just chroot
	// Mount the rootfs.ext4 temporarily to inject the container.cmd file
	jailerRoot := filepath.Join(c.jailerConfig.ChrootBaseDir, "firecracker", vmID, "root")
	rootfsPath := filepath.Join(jailerRoot, "rootfs.ext4")

	// Create temporary mount point
	tmpMount := filepath.Join("/tmp", "rootfs-mount-"+vmID)
	if err := os.MkdirAll(tmpMount, 0o755); err != nil {
		return fmt.Errorf("failed to create temp mount dir: %w", err)
	}
	defer os.RemoveAll(tmpMount)

	// Mount the rootfs.ext4
	mountCmd := exec.Command("mount", "-o", "loop", rootfsPath, tmpMount)
	if err := mountCmd.Run(); err != nil {
		return fmt.Errorf("failed to mount rootfs: %w", err)
	}
	defer func() {
		umountCmd := exec.Command("umount", tmpMount)
		umountCmd.Run()
	}()

	// Write container.cmd into the mounted filesystem
	containerCmdPath := filepath.Join(tmpMount, "container.cmd")
	if err := os.WriteFile(containerCmdPath, cmdJSON, 0o600); err != nil {
		return fmt.Errorf("failed to write container.cmd to rootfs: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "created container.cmd file",
		slog.String("vm_id", vmID),
		slog.String("path", containerCmdPath),
		slog.String("command", string(cmdJSON)),
	)

	return nil
}

// copyMetadataFilesForAssets copies metadata files alongside rootfs assets when using asset manager
func (c *Client) copyMetadataFilesForAssets(ctx context.Context, vmID string, config *metaldv1.VmConfig, preparedPaths map[string]string, jailerRoot string) error {
	// When using asset manager, only rootfs files are copied, but we need metadata files too
	// This function finds the original metadata files and copies them to the jailer root

	for _, disk := range config.GetStorage() {
		if !disk.GetIsRootDevice() || disk.GetPath() == "" {
			continue
		}

		// Find the original rootfs path before asset preparation
		originalRootfsPath := disk.GetPath()

		// Check if this disk was replaced by an asset
		var preparedRootfsPath string
		for _, path := range preparedPaths {
			if strings.HasSuffix(path, ".ext4") || strings.HasSuffix(path, ".img") {
				preparedRootfsPath = path
				break
			}
		}

		if preparedRootfsPath == "" {
			// No rootfs asset found, skip metadata copying
			continue
		}

		// Look for metadata file alongside the original rootfs
		originalDir := filepath.Dir(originalRootfsPath)
		originalBaseName := strings.TrimSuffix(filepath.Base(originalRootfsPath), filepath.Ext(originalRootfsPath))
		metadataSrcPath := filepath.Join(originalDir, originalBaseName+".metadata.json")

		// Check if metadata file exists
		if _, err := os.Stat(metadataSrcPath); os.IsNotExist(err) {
			c.logger.LogAttrs(ctx, slog.LevelDebug, "no metadata file found for asset",
				slog.String("vm_id", vmID),
				slog.String("original_rootfs", originalRootfsPath),
				slog.String("expected_metadata", metadataSrcPath),
			)
			continue
		}

		// Copy metadata file to jailer root with the same base name as the prepared rootfs
		preparedBaseName := strings.TrimSuffix(filepath.Base(preparedRootfsPath), filepath.Ext(preparedRootfsPath))
		metadataDstPath := filepath.Join(jailerRoot, preparedBaseName+".metadata.json")

		if err := copyFileWithOwnership(metadataSrcPath, metadataDstPath, int(c.jailerConfig.UID), int(c.jailerConfig.GID)); err != nil {
			c.logger.WarnContext(ctx, "failed to copy metadata file",
				slog.String("vm_id", vmID),
				slog.String("src", metadataSrcPath),
				slog.String("dst", metadataDstPath),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("failed to copy metadata file %s: %w", metadataSrcPath, err)
		}

		c.logger.InfoContext(ctx, "copied metadata file for asset",
			slog.String("vm_id", vmID),
			slog.String("src", metadataSrcPath),
			slog.String("dst", metadataDstPath),
		)
	}

	return nil
}

// copyFileWithOwnership copies files with ownership
func copyFileWithOwnership(src, dst string, uid, gid int) error {
	// Use cp command to handle large files efficiently
	cmd := exec.Command("cp", "-f", src, dst)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cp command failed: %w, output: %s", err, output)
	}

	// Set permissions
	if err := os.Chmod(dst, 0o644); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}

	// Set ownership
	if err := os.Chown(dst, uid, gid); err != nil {
		// Log but don't fail - might work anyway
		return nil
	}

	return nil
}

// validateIPAddress validates an IP address to prevent command injection
func validateIPAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

// validatePortNumber validates a port number to prevent command injection
func validatePortNumber(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number: %d, must be between 1-65535", port)
	}
	return nil
}
