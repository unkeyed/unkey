package assetmanager

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/unkeyed/unkey/go/apps/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1/assetmanagerdv1connect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client provides access to the assetmanagerd service
type Client struct {
	client   assetmanagerdv1connect.AssetManagerServiceClient
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
	client := assetmanagerdv1connect.NewAssetManagerServiceClient(
		httpClient,
		cfg.AssetManager.Endpoint,
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

// RegisterBuildArtifactWithID uploads and registers a successfully built artifact with a specific asset ID
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

	// Prepare labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["build_id"] = buildID
	labels["created_by"] = "builderd"

	// Create upload metadata
	metadata := &assetv1.UploadAssetMetadata{
		Name:        filepath.Base(artifactPath),
		Type:        assetType,
		SizeBytes:   fileInfo.Size(),
		Labels:      labels,
		CreatedBy:   "builderd",
		BuildId:     buildID,
		SourceImage: labels["docker_image"], // Optional, from build metadata
		Id:          assetID,                // Optional, use pre-generated ID if provided
	}

	// Upload asset via streaming API
	// AIDEV-NOTE: This properly uploads the file to assetmanagerd's storage and registers it
	stream := c.client.UploadAsset(ctx)

	// Send metadata first
	metadataReq := &assetv1.UploadAssetRequest{
		Data: &assetv1.UploadAssetRequest_Metadata{
			Metadata: metadata,
		},
	}
	if err := stream.Send(metadataReq); err != nil {
		return "", fmt.Errorf("failed to send metadata: %w", err)
	}

	// Open file for streaming
	file, err := os.Open(artifactPath)
	if err != nil {
		return "", fmt.Errorf("failed to open artifact file: %w", err)
	}
	defer file.Close()

	// Stream file in chunks
	const chunkSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read file chunk: %w", err)
		}
		if n == 0 {
			break
		}

		chunkReq := &assetv1.UploadAssetRequest{
			Data: &assetv1.UploadAssetRequest_Chunk{
				Chunk: buffer[:n],
			},
		}
		if err := stream.Send(chunkReq); err != nil {
			return "", fmt.Errorf("failed to send chunk: %w", err)
		}
	}

	// Close and receive response
	resp, err := stream.CloseAndReceive()
	if err != nil {
		return "", fmt.Errorf("failed to upload asset: %w", err)
	}

	c.logger.InfoContext(ctx, "uploaded and registered build artifact with assetmanagerd",
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
