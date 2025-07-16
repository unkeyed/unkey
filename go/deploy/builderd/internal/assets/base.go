package assets

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/assetmanager"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
)

// BaseAssetManager handles initialization and registration of base VM assets
type BaseAssetManager struct {
	logger      *slog.Logger
	config      *config.Config
	assetClient *assetmanager.Client
	storageDir  string
}

// BaseAsset represents a base asset that needs to be downloaded and registered
type BaseAsset struct {
	Name        string
	URL         string
	Type        assetv1.AssetType
	Description string
	Labels      map[string]string
}

// NewBaseAssetManager creates a new base asset manager
func NewBaseAssetManager(logger *slog.Logger, cfg *config.Config, assetClient *assetmanager.Client) *BaseAssetManager {
	return &BaseAssetManager{
		logger:      logger.With("component", "base-asset-manager"),
		config:      cfg,
		assetClient: assetClient,
		storageDir:  cfg.Builder.RootfsOutputDir,
	}
}

// InitializeBaseAssets ensures all required base assets are available
func (m *BaseAssetManager) InitializeBaseAssets(ctx context.Context) error {
	// AIDEV-NOTE: Base assets required for VM creation
	// These are downloaded from Firecracker quickstart guide if not already available
	baseAssets := []BaseAsset{
		{
			Name:        "vmlinux",
			URL:         "https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin",
			Type:        assetv1.AssetType_ASSET_TYPE_KERNEL,
			Description: "Firecracker x86_64 kernel",
			Labels: map[string]string{
				"architecture": "x86_64",
				"source":       "firecracker-quickstart",
				"asset_type":   "kernel",
			},
		},
		{
			Name:        "rootfs.ext4",
			URL:         "https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/rootfs/bionic.rootfs.ext4",
			Type:        assetv1.AssetType_ASSET_TYPE_ROOTFS,
			Description: "Ubuntu Bionic base rootfs",
			Labels: map[string]string{
				"architecture": "x86_64",
				"source":       "firecracker-quickstart",
				"asset_type":   "rootfs",
				"os":           "ubuntu",
				"version":      "bionic",
			},
		},
	}

	for _, asset := range baseAssets {
		if err := m.ensureAssetAvailable(ctx, asset); err != nil {
			return fmt.Errorf("failed to ensure asset %s is available: %w", asset.Name, err)
		}
	}

	m.logger.InfoContext(ctx, "base assets initialization completed")
	return nil
}

// ensureAssetAvailable checks if an asset exists and is registered, downloads and registers if needed
func (m *BaseAssetManager) ensureAssetAvailable(ctx context.Context, asset BaseAsset) error {
	// Check if asset is already registered
	if m.assetClient != nil {
		exists, err := m.checkAssetRegistered(ctx, asset)
		if err != nil {
			m.logger.WarnContext(ctx, "failed to check asset registration, proceeding with download",
				"asset", asset.Name,
				"error", err,
			)
		} else if exists {
			m.logger.InfoContext(ctx, "asset already registered",
				"asset", asset.Name,
			)
			return nil
		}
	}

	// Download asset if not present locally
	localPath := filepath.Join(m.storageDir, "base", asset.Name)
	if err := m.downloadAsset(ctx, asset, localPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Register with assetmanagerd if enabled
	if m.assetClient != nil {
		if err := m.registerAsset(ctx, asset, localPath); err != nil {
			return fmt.Errorf("failed to register asset: %w", err)
		}
	}

	return nil
}

// checkAssetRegistered checks if an asset is already registered in assetmanagerd
func (m *BaseAssetManager) checkAssetRegistered(ctx context.Context, asset BaseAsset) (bool, error) {
	// TODO: Implement asset query to check if base asset already exists
	// For now, return false to always download/register
	return false, nil
}

// downloadAsset downloads an asset from URL to local path
func (m *BaseAssetManager) downloadAsset(ctx context.Context, asset BaseAsset, localPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(localPath); err == nil {
		m.logger.InfoContext(ctx, "asset already exists locally",
			"asset", asset.Name,
			"path", localPath,
		)
		return nil
	}

	m.logger.InfoContext(ctx, "downloading asset",
		"asset", asset.Name,
		"url", asset.URL,
		"path", localPath,
	)

	// Download with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download asset: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tmpPath := localPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpPath)

	// Copy with progress
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write asset: %w", err)
	}
	tmpFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, localPath); err != nil {
		return fmt.Errorf("failed to finalize asset: %w", err)
	}

	m.logger.InfoContext(ctx, "asset downloaded successfully",
		"asset", asset.Name,
		"size_bytes", written,
		"path", localPath,
	)

	return nil
}

// registerAsset registers an asset with assetmanagerd
func (m *BaseAssetManager) registerAsset(ctx context.Context, asset BaseAsset, localPath string) error {
	// Get file info
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat asset file: %w", err)
	}

	// Calculate checksum
	checksum, err := m.calculateChecksum(localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Prepare labels
	labels := make(map[string]string)
	for k, v := range asset.Labels {
		labels[k] = v
	}
	labels["created_by"] = "builderd"
	labels["customer_id"] = "system"
	labels["tenant_id"] = "system"

	// Get relative path within storage directory
	relPath, err := filepath.Rel(m.storageDir, localPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Register via assetmanager client
	assetID, err := m.assetClient.RegisterBuildArtifact(ctx, "base-assets", localPath, asset.Type, labels)
	if err != nil {
		return fmt.Errorf("failed to register asset: %w", err)
	}

	m.logger.InfoContext(ctx, "asset registered successfully",
		"asset", asset.Name,
		"asset_id", assetID,
		"location", relPath,
		"size_bytes", fileInfo.Size(),
		"checksum", checksum,
	)

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (m *BaseAssetManager) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
