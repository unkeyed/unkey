package assetmanager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1/assetmanagerdv1connect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client provides access to assetmanagerd services
type Client interface {
	// ListAssets returns available assets with optional filtering
	ListAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string) ([]*assetv1.Asset, error)

	// QueryAssets returns available assets with automatic build triggering if not found
	QueryAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string, buildOptions *assetv1.BuildOptions) (*assetv1.QueryAssetsResponse, error)

	// PrepareAssets stages assets for a specific VM in the target path
	PrepareAssets(ctx context.Context, assetIDs []string, targetPath string, vmID string) (map[string]string, error)

	// AcquireAsset marks an asset as in-use by a VM
	AcquireAsset(ctx context.Context, assetID string, vmID string) (string, error)

	// ReleaseAsset releases an asset reference
	ReleaseAsset(ctx context.Context, leaseID string) error
}

// client implements the Client interface
type client struct {
	assetClient assetmanagerdv1connect.AssetManagerServiceClient
	logger      *slog.Logger
}

// NewClient creates a new assetmanagerd client
func NewClient(cfg *config.AssetManagerConfig, logger *slog.Logger) (Client, error) {
	if !cfg.Enabled {
		return &noopClient{}, nil
	}

	// Create HTTP client with timeouts and OpenTelemetry instrumentation
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Create Connect client with default client interceptors plus custom logging
	// AIDEV-NOTE: Using shared client interceptors for consistency across services
	clientInterceptors := interceptors.NewDefaultClientInterceptors("metald", logger)
	// Add custom logging and debug interceptors
	clientInterceptors = append(clientInterceptors,
		loggingInterceptor(logger),
		observability.DebugInterceptor(logger, "assetmanager"),
	)

	// Convert UnaryInterceptorFunc to Interceptor
	var interceptorList []connect.Interceptor
	for _, interceptor := range clientInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	assetClient := assetmanagerdv1connect.NewAssetManagerServiceClient(
		httpClient,
		cfg.Endpoint,
		connect.WithInterceptors(interceptorList...),
	)

	return &client{
		assetClient: assetClient,
		logger:      logger.With(slog.String("component", "assetmanager-client")),
	}, nil
}

// NewClientWithHTTP creates a new assetmanagerd client with a custom HTTP client (for TLS)
func NewClientWithHTTP(cfg *config.AssetManagerConfig, logger *slog.Logger, httpClient *http.Client) (Client, error) {
	if !cfg.Enabled {
		return &noopClient{}, nil
	}

	// Use provided HTTP client which may have TLS configuration
	// AIDEV-NOTE: Using shared client interceptors for consistency across services
	clientInterceptors := interceptors.NewDefaultClientInterceptors("metald", logger)
	// Add custom logging and debug interceptors
	clientInterceptors = append(clientInterceptors,
		loggingInterceptor(logger),
		observability.DebugInterceptor(logger, "assetmanager"),
	)

	// Convert UnaryInterceptorFunc to Interceptor
	var interceptorList []connect.Interceptor
	for _, interceptor := range clientInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	assetClient := assetmanagerdv1connect.NewAssetManagerServiceClient(
		httpClient,
		cfg.Endpoint,
		connect.WithInterceptors(interceptorList...),
	)

	return &client{
		assetClient: assetClient,
		logger:      logger.With(slog.String("component", "assetmanager-client")),
	}, nil
}

// ListAssets returns available assets with optional filtering
func (c *client) ListAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string) ([]*assetv1.Asset, error) {
	// AIDEV-NOTE: Pagination is not implemented in this initial version
	// For production use, implement pagination handling based on expected asset counts

	//exhaustruct:ignore
	req := &assetv1.ListAssetsRequest{
		Type:          assetType,
		Status:        assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
		LabelSelector: labels,
		PageSize:      1000, // Reasonable default for initial implementation
	}

	resp, err := c.assetClient.ListAssets(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.LogAttrs(ctx, slog.LevelError, "assetmanager connection error",
				slog.String("error", err.Error()),
				slog.String("code", connectErr.Code().String()),
				slog.String("message", connectErr.Message()),
				slog.String("asset_type", assetType.String()),
				slog.String("operation", "ListAssets"),
			)
		} else {
			c.logger.LogAttrs(ctx, slog.LevelError, "failed to list assets",
				slog.String("error", err.Error()),
				slog.String("asset_type", assetType.String()),
				slog.String("operation", "ListAssets"),
			)
		}
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "listed assets",
		slog.Int("count", len(resp.Msg.GetAssets())),
		slog.String("asset_type", assetType.String()),
	)

	return resp.Msg.GetAssets(), nil
}

// QueryAssets returns available assets with automatic build triggering if not found
func (c *client) QueryAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string, buildOptions *assetv1.BuildOptions) (*assetv1.QueryAssetsResponse, error) {
	// AIDEV-NOTE: This method supports automatic asset building when assets don't exist
	// It's the key integration point for the metald → assetmanagerd → builderd workflow

	//exhaustruct:ignore
	req := &assetv1.QueryAssetsRequest{
		Type:          assetType,
		LabelSelector: labels,
		PageSize:      1000, // Reasonable default for initial implementation
		BuildOptions:  buildOptions,
	}

	// Only filter by AVAILABLE status if we're not doing automatic builds
	// Otherwise we might miss PENDING/BUILDING assets and trigger duplicate builds
	if buildOptions == nil || !buildOptions.GetEnableAutoBuild() {
		req.Status = assetv1.AssetStatus_ASSET_STATUS_AVAILABLE
	}

	resp, err := c.assetClient.QueryAssets(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.LogAttrs(ctx, slog.LevelError, "assetmanager connection error",
				slog.String("error", err.Error()),
				slog.String("code", connectErr.Code().String()),
				slog.String("message", connectErr.Message()),
				slog.String("asset_type", assetType.String()),
				slog.String("operation", "QueryAssets"),
			)
		} else {
			c.logger.LogAttrs(ctx, slog.LevelError, "failed to query assets",
				slog.String("error", err.Error()),
				slog.String("asset_type", assetType.String()),
				slog.String("operation", "QueryAssets"),
			)
		}
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "queried assets",
		slog.Int("asset_count", len(resp.Msg.GetAssets())),
		slog.Int("builds_triggered", len(resp.Msg.GetTriggeredBuilds())),
		slog.String("asset_type", assetType.String()),
	)

	// Log any triggered builds
	for _, build := range resp.Msg.GetTriggeredBuilds() {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "build triggered for missing asset",
			slog.String("build_id", build.GetBuildId()),
			slog.String("docker_image", build.GetDockerImage()),
			slog.String("status", build.GetStatus()),
			slog.String("asset_id", build.GetAssetId()),
		)
	}

	return resp.Msg, nil
}

// PrepareAssets stages assets for a specific VM in the target path
func (c *client) PrepareAssets(ctx context.Context, assetIDs []string, targetPath string, vmID string) (map[string]string, error) {
	req := &assetv1.PrepareAssetsRequest{
		AssetIds:    assetIDs,
		TargetPath:  targetPath,
		PreparedFor: vmID,
	}

	resp, err := c.assetClient.PrepareAssets(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.LogAttrs(ctx, slog.LevelError, "assetmanager connection error",
				slog.String("error", err.Error()),
				slog.String("code", connectErr.Code().String()),
				slog.String("message", connectErr.Message()),
				slog.String("vm_id", vmID),
				slog.String("target_path", targetPath),
				slog.String("operation", "PrepareAssets"),
				slog.Int("asset_count", len(assetIDs)),
			)
		} else {
			c.logger.LogAttrs(ctx, slog.LevelError, "failed to prepare assets",
				slog.String("error", err.Error()),
				slog.String("vm_id", vmID),
				slog.String("target_path", targetPath),
				slog.String("operation", "PrepareAssets"),
				slog.Int("asset_count", len(assetIDs)),
			)
		}
		return nil, fmt.Errorf("failed to prepare assets: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "prepared assets for VM",
		slog.String("vm_id", vmID),
		slog.Int("asset_count", len(resp.Msg.GetAssetPaths())),
	)

	return resp.Msg.GetAssetPaths(), nil
}

// AcquireAsset marks an asset as in-use by a VM
func (c *client) AcquireAsset(ctx context.Context, assetID string, vmID string) (string, error) {
	req := &assetv1.AcquireAssetRequest{
		AssetId:    assetID,
		AcquiredBy: vmID,
		TtlSeconds: 86400, // 24 hours default TTL
	}

	resp, err := c.assetClient.AcquireAsset(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.LogAttrs(ctx, slog.LevelError, "assetmanager connection error",
				slog.String("error", err.Error()),
				slog.String("code", connectErr.Code().String()),
				slog.String("message", connectErr.Message()),
				slog.String("asset_id", assetID),
				slog.String("vm_id", vmID),
				slog.String("operation", "AcquireAsset"),
			)
		} else {
			c.logger.LogAttrs(ctx, slog.LevelError, "failed to acquire asset",
				slog.String("error", err.Error()),
				slog.String("asset_id", assetID),
				slog.String("vm_id", vmID),
				slog.String("operation", "AcquireAsset"),
			)
		}
		return "", fmt.Errorf("failed to acquire asset: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "acquired asset",
		slog.String("asset_id", assetID),
		slog.String("vm_id", vmID),
		slog.String("lease_id", resp.Msg.GetLeaseId()),
	)

	return resp.Msg.GetLeaseId(), nil
}

// ReleaseAsset releases an asset reference
func (c *client) ReleaseAsset(ctx context.Context, leaseID string) error {
	req := &assetv1.ReleaseAssetRequest{
		LeaseId: leaseID,
	}

	_, err := c.assetClient.ReleaseAsset(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.LogAttrs(ctx, slog.LevelError, "assetmanager connection error",
				slog.String("error", err.Error()),
				slog.String("code", connectErr.Code().String()),
				slog.String("message", connectErr.Message()),
				slog.String("lease_id", leaseID),
				slog.String("operation", "ReleaseAsset"),
			)
		} else {
			c.logger.LogAttrs(ctx, slog.LevelError, "failed to release asset",
				slog.String("error", err.Error()),
				slog.String("lease_id", leaseID),
				slog.String("operation", "ReleaseAsset"),
			)
		}
		return fmt.Errorf("failed to release asset: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "released asset",
		slog.String("lease_id", leaseID),
	)

	return nil
}

// noopClient is used when assetmanagerd integration is disabled
type noopClient struct{}

func (n *noopClient) ListAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string) ([]*assetv1.Asset, error) {
	// Return empty list when disabled
	return []*assetv1.Asset{}, nil
}

func (n *noopClient) QueryAssets(ctx context.Context, assetType assetv1.AssetType, labels map[string]string, buildOptions *assetv1.BuildOptions) (*assetv1.QueryAssetsResponse, error) {
	// Return empty response when disabled
	return &assetv1.QueryAssetsResponse{
		Assets: []*assetv1.Asset{},
	}, nil
}

func (n *noopClient) PrepareAssets(ctx context.Context, assetIDs []string, targetPath string, vmID string) (map[string]string, error) {
	// Return empty map when disabled
	return map[string]string{}, nil
}

func (n *noopClient) AcquireAsset(ctx context.Context, assetID string, vmID string) (string, error) {
	// Return empty lease ID when disabled
	return "", nil
}

func (n *noopClient) ReleaseAsset(ctx context.Context, leaseID string) error {
	// No-op when disabled
	return nil
}

// loggingInterceptor provides basic logging for RPC calls
func loggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()

			// Execute request
			resp, err := next(ctx, req)

			// Log result
			duration := time.Since(start)
			if err != nil {
				// AIDEV-NOTE: Enhanced debug logging for RPC errors
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					logger.LogAttrs(ctx, slog.LevelError, "assetmanager rpc connection error",
						slog.String("procedure", req.Spec().Procedure),
						slog.Duration("duration", duration),
						slog.String("error", err.Error()),
						slog.String("code", connectErr.Code().String()),
						slog.String("details", connectErr.Message()),
					)
				} else {
					logger.LogAttrs(ctx, slog.LevelError, "assetmanager rpc error",
						slog.String("procedure", req.Spec().Procedure),
						slog.Duration("duration", duration),
						slog.String("error", err.Error()),
					)
				}
			} else {
				logger.LogAttrs(ctx, slog.LevelDebug, "assetmanager rpc success",
					slog.String("procedure", req.Spec().Procedure),
					slog.Duration("duration", duration),
				)
			}

			return resp, err
		}
	}
}
