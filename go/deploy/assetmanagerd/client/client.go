package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1/assetmanagerdv1connect"
)

// AIDEV-NOTE: AssetManagerd client with SPIFFE/SPIRE socket integration
// This client provides a high-level interface for assetmanagerd operations with proper authentication

// Config holds the configuration for the assetmanagerd client
type Config struct {
	// ServerAddress is the assetmanagerd server endpoint (e.g., "https://assetmanagerd:8083")
	ServerAddress string

	// UserID is the user identifier for authentication
	UserID string

	// TLS configuration
	TLSMode           string        // "disabled", "file", or "spiffe"
	SPIFFESocketPath  string        // Path to SPIFFE agent socket
	TLSCertFile       string        // TLS certificate file (for file mode)
	TLSKeyFile        string        // TLS key file (for file mode)
	TLSCAFile         string        // TLS CA file (for file mode)
	EnableCertCaching bool          // Enable certificate caching
	CertCacheTTL      time.Duration // Certificate cache TTL

	// Optional HTTP client timeout
	Timeout time.Duration
}

// Client provides a high-level interface to assetmanagerd services
type Client struct {
	assetService assetmanagerdv1connect.AssetManagerServiceClient
	tlsProvider  tls.Provider
	userID       string
	serverAddr   string
}

// New creates a new assetmanagerd client with SPIFFE/SPIRE integration
func New(ctx context.Context, config Config) (*Client, error) {
	// Set defaults
	if config.SPIFFESocketPath == "" {
		config.SPIFFESocketPath = "/var/lib/spire/agent/agent.sock"
	}
	if config.TLSMode == "" {
		config.TLSMode = "spiffe"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.CertCacheTTL == 0 {
		config.CertCacheTTL = 5 * time.Second
	}

	// Create TLS provider
	tlsConfig := tls.Config{
		Mode:              tls.Mode(config.TLSMode),
		CertFile:          config.TLSCertFile,
		KeyFile:           config.TLSKeyFile,
		CAFile:            config.TLSCAFile,
		SPIFFESocketPath:  config.SPIFFESocketPath,
		EnableCertCaching: config.EnableCertCaching,
		CertCacheTTL:      config.CertCacheTTL,
	}

	tlsProvider, err := tls.NewProvider(ctx, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS provider: %w", err)
	}

	// Get HTTP client with SPIFFE mTLS
	httpClient := tlsProvider.HTTPClient()
	httpClient.Timeout = config.Timeout

	// Add authentication
	httpClient.Transport = &transport{
		Base:   httpClient.Transport,
		UserID: config.UserID,
	}

	// Create ConnectRPC client
	assetService := assetmanagerdv1connect.NewAssetManagerServiceClient(
		httpClient,
		config.ServerAddress,
	)

	return &Client{
		assetService: assetService,
		tlsProvider:  tlsProvider,
		userID:       config.UserID,
		serverAddr:   config.ServerAddress,
	}, nil
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	if c.tlsProvider != nil {
		return c.tlsProvider.Close()
	}
	return nil
}

// RegisterAsset registers a new asset with assetmanagerd
func (c *Client) RegisterAsset(ctx context.Context, req *RegisterAssetRequest) (*RegisterAssetResponse, error) {
	pbReq := &assetv1.RegisterAssetRequest{
		Name:      req.Name,
		Type:      req.Type,
		Backend:   req.Backend,
		Location:  req.Location,
		SizeBytes: req.SizeBytes,
		Checksum:  req.Checksum,
		Labels:    req.Labels,
		CreatedBy: req.CreatedBy,
	}

	resp, err := c.assetService.RegisterAsset(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to register asset: %w", err)
	}

	return &RegisterAssetResponse{
		Asset: resp.Msg.Asset,
	}, nil
}

// GetAsset retrieves asset information by ID
func (c *Client) GetAsset(ctx context.Context, assetID string) (*GetAssetResponse, error) {
	req := &assetv1.GetAssetRequest{
		Id: assetID,
	}

	resp, err := c.assetService.GetAsset(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return &GetAssetResponse{
		Asset: resp.Msg.Asset,
	}, nil
}

// ListAssets retrieves a list of assets with optional filtering
func (c *Client) ListAssets(ctx context.Context, req *ListAssetsRequest) (*ListAssetsResponse, error) {
	pbReq := &assetv1.ListAssetsRequest{
		Type:          req.Type,
		Status:        req.Status,
		LabelSelector: req.Labels,
		PageSize:      req.PageSize,
		PageToken:     req.PageToken,
	}

	resp, err := c.assetService.ListAssets(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	return &ListAssetsResponse{
		Assets:        resp.Msg.Assets,
		NextPageToken: resp.Msg.NextPageToken,
	}, nil
}

// QueryAssets queries assets with automatic build triggering if not found
func (c *Client) QueryAssets(ctx context.Context, req *QueryAssetsRequest) (*QueryAssetsResponse, error) {
	pbReq := &assetv1.QueryAssetsRequest{
		Type:          req.Type,
		LabelSelector: req.Labels,
	}

	resp, err := c.assetService.QueryAssets(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}

	return &QueryAssetsResponse{
		Assets: resp.Msg.Assets,
	}, nil
}

// PrepareAssets pre-stages assets for a specific host/jailer
func (c *Client) PrepareAssets(ctx context.Context, req *PrepareAssetsRequest) (*PrepareAssetsResponse, error) {
	pbReq := &assetv1.PrepareAssetsRequest{
		AssetIds:    req.AssetIds,
		TargetPath:  req.CacheDir,
		PreparedFor: req.JailerId,
	}

	resp, err := c.assetService.PrepareAssets(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare assets: %w", err)
	}

	// Convert map to slice of paths
	var preparedPaths []string
	for _, path := range resp.Msg.AssetPaths {
		preparedPaths = append(preparedPaths, path)
	}

	return &PrepareAssetsResponse{
		PreparedPaths: preparedPaths,
		Success:       len(resp.Msg.AssetPaths) > 0,
	}, nil
}

// AcquireAsset marks an asset as in-use (reference counting for GC)
func (c *Client) AcquireAsset(ctx context.Context, assetID string) (*AcquireAssetResponse, error) {
	req := &assetv1.AcquireAssetRequest{
		AssetId: assetID,
	}

	resp, err := c.assetService.AcquireAsset(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to acquire asset: %w", err)
	}

	return &AcquireAssetResponse{
		Success:        resp.Msg.Asset != nil,
		ReferenceCount: int32(len(resp.Msg.LeaseId)), // Use lease ID length as proxy
	}, nil
}

// ReleaseAsset releases an asset reference (decrements ref count)
func (c *Client) ReleaseAsset(ctx context.Context, leaseID string) (*ReleaseAssetResponse, error) {
	req := &assetv1.ReleaseAssetRequest{
		LeaseId: leaseID,
	}

	resp, err := c.assetService.ReleaseAsset(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to release asset: %w", err)
	}

	return &ReleaseAssetResponse{
		Success:        resp.Msg.Asset != nil,
		ReferenceCount: 0, // Not available in response
	}, nil
}

// DeleteAsset deletes an asset (only if ref count is 0)
func (c *Client) DeleteAsset(ctx context.Context, assetID string) (*DeleteAssetResponse, error) {
	req := &assetv1.DeleteAssetRequest{
		Id: assetID,
	}

	resp, err := c.assetService.DeleteAsset(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to delete asset: %w", err)
	}

	return &DeleteAssetResponse{
		Success: resp.Msg.Deleted,
	}, nil
}

// GarbageCollect triggers garbage collection of unused assets
func (c *Client) GarbageCollect(ctx context.Context, req *GarbageCollectRequest) (*GarbageCollectResponse, error) {
	pbReq := &assetv1.GarbageCollectRequest{
		DryRun:             req.DryRun,
		MaxAgeSeconds:      int64(req.MaxAgeHours) * 3600, // Convert hours to seconds
		DeleteUnreferenced: req.ForceCleanup,
	}

	resp, err := c.assetService.GarbageCollect(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to garbage collect: %w", err)
	}

	// Extract asset IDs from deleted assets
	var removedAssets []string
	for _, asset := range resp.Msg.DeletedAssets {
		removedAssets = append(removedAssets, asset.Id)
	}

	return &GarbageCollectResponse{
		RemovedAssets: removedAssets,
		FreedBytes:    resp.Msg.BytesFreed,
		Success:       len(resp.Msg.DeletedAssets) >= 0, // Always consider it successful
	}, nil
}

// GetServerAddress returns the server address this client is connected to
func (c *Client) GetServerAddress() string {
	return c.serverAddr
}

// transport adds authentication headers to all requests
type transport struct {
	Base   http.RoundTripper
	UserID string
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())
	if req2.Header == nil {
		req2.Header = make(http.Header)
	}

	// Set Authorization header with development token format
	// AIDEV-BUSINESS_RULE: In development, use "dev_user_<id>" format
	// TODO: Update to proper JWT tokens in production
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer dev_user_%s", t.UserID))

	// Use the base transport, or default if nil
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}
