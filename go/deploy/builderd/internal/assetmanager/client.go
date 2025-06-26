package assetmanager

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	assetv1 "github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1/assetv1connect"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client provides access to the assetmanagerd service
type Client struct {
	client   assetv1connect.AssetManagerServiceClient
	logger   *slog.Logger
	enabled  bool
	endpoint string
}

// NewClient creates a new assetmanagerd client
func NewClient(cfg *config.Config, logger *slog.Logger, tlsProvider tls.Provider) (*Client, error) {
	if !cfg.AssetManager.Enabled {
		logger.Info("assetmanagerd integration disabled")
		return &Client{
			client:   nil,
			logger:   logger,
			enabled:  false,
			endpoint: "",
		}, nil
	}

	// Get HTTP client with TLS configuration
	httpClient := tlsProvider.HTTPClient()

	// Wrap with OpenTelemetry instrumentation for trace propagation
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)

	// Create Connect client
	client := assetv1connect.NewAssetManagerServiceClient(
		httpClient,
		cfg.AssetManager.Endpoint,
		connect.WithGRPC(),
	)

	logger.Info("initialized assetmanagerd client",
		slog.String("endpoint", cfg.AssetManager.Endpoint),
	)

	return &Client{
		client:   client,
		logger:   logger,
		enabled:  true,
		endpoint: cfg.AssetManager.Endpoint,
	}, nil
}

// RegisterBuildArtifact registers a successfully built artifact with assetmanagerd
// AIDEV-NOTE: This is called after a successful build to make the artifact available for VM creation
func (c *Client) RegisterBuildArtifact(ctx context.Context, buildID, artifactPath string, assetType assetv1.AssetType, labels map[string]string) (string, error) {
	return c.RegisterBuildArtifactWithID(ctx, buildID, artifactPath, assetType, labels, "")
}

// RegisterBuildArtifactWithID registers a successfully built artifact with a specific asset ID
func (c *Client) RegisterBuildArtifactWithID(ctx context.Context, buildID, artifactPath string, assetType assetv1.AssetType, labels map[string]string, assetID string) (string, error) {
	if !c.enabled {
		c.logger.DebugContext(ctx, "assetmanagerd integration disabled, skipping artifact registration")
		return "", nil
	}

	// Get file info
	fileInfo, err := os.Stat(artifactPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat artifact file: %w", err)
	}

	// Calculate checksum
	checksum, err := c.calculateChecksum(artifactPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Prepare labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["build_id"] = buildID
	labels["created_by"] = "builderd"

	// Create register request
	// AIDEV-NOTE: Location should be relative to storage backend's base path
	req := &assetv1.RegisterAssetRequest{
		Name:         filepath.Base(artifactPath),
		Type:         assetType,
		Backend:      assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
		Location:     filepath.Base(artifactPath), // Just the filename, not full path
		SizeBytes:    fileInfo.Size(),
		Checksum:     checksum,
		Labels:       labels,
		CreatedBy:    "builderd",
		BuildId:      buildID,
		SourceImage:  labels["docker_image"], // Optional, from build metadata
		Id:           assetID, // Optional, use pre-generated ID if provided
	}

	// Register with assetmanagerd
	resp, err := c.client.RegisterAsset(ctx, connect.NewRequest(req))
	if err != nil {
		return "", fmt.Errorf("failed to register asset: %w", err)
	}

	c.logger.InfoContext(ctx, "registered build artifact with assetmanagerd",
		slog.String("asset_id", resp.Msg.GetAsset().GetId()),
		slog.String("build_id", buildID),
		slog.String("artifact_path", artifactPath),
		slog.String("asset_type", assetType.String()),
	)

	return resp.Msg.GetAsset().GetId(), nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (c *Client) calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
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

// IsEnabled returns whether assetmanagerd integration is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled
}