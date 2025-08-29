package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1/builderdv1connect"
)

// AIDEV-NOTE: Builderd client with SPIFFE/SPIRE socket integration
// This client provides a high-level interface for builderd operations with proper authentication

// Config holds the configuration for the builderd client
type Config struct {
	// ServerAddress is the builderd server endpoint (e.g., "https://builderd:8082")
	ServerAddress string

	// UserID is the user identifier for authentication
	UserID string

	// TenantID is the tenant identifier for data scoping
	TenantID string

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

// Client provides a high-level interface to builderd services
type Client struct {
	builderService builderdv1connect.BuilderServiceClient
	tlsProvider    tls.Provider
	userID         string
	tenantID       string
	serverAddr     string
}

// New creates a new builderd client with SPIFFE/SPIRE integration
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

	// Add authentication and tenant isolation transport
	httpClient.Transport = &tenantTransport{
		Base:     httpClient.Transport,
		UserID:   config.UserID,
		TenantID: config.TenantID,
	}

	// Create ConnectRPC client
	builderService := builderdv1connect.NewBuilderServiceClient(
		httpClient,
		config.ServerAddress,
	)

	return &Client{
		builderService: builderService,
		tlsProvider:    tlsProvider,
		userID:         config.UserID,
		tenantID:       config.TenantID,
		serverAddr:     config.ServerAddress,
	}, nil
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	if c.tlsProvider != nil {
		return c.tlsProvider.Close()
	}
	return nil
}

// CreateBuild creates a new build job
func (c *Client) CreateBuild(ctx context.Context, req *CreateBuildRequest) (*CreateBuildResponse, error) {
	pbReq := &builderv1.CreateBuildRequest{
		Config: req.Config,
	}

	resp, err := c.builderService.CreateBuild(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create build: %w", err)
	}

	return &CreateBuildResponse{
		BuildID:    resp.Msg.BuildId,
		State:      resp.Msg.State,
		CreatedAt:  resp.Msg.CreatedAt,
		RootfsPath: resp.Msg.RootfsPath,
	}, nil
}

// GetBuild retrieves build status and progress
func (c *Client) GetBuild(ctx context.Context, req *GetBuildRequest) (*GetBuildResponse, error) {
	pbReq := &builderv1.GetBuildRequest{
		BuildId:  req.BuildID,
		TenantId: req.TenantID,
	}

	resp, err := c.builderService.GetBuild(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to get build: %w", err)
	}

	return &GetBuildResponse{
		Build: resp.Msg.Build,
	}, nil
}

// ListBuilds lists builds with filtering (tenant-scoped)
func (c *Client) ListBuilds(ctx context.Context, req *ListBuildsRequest) (*ListBuildsResponse, error) {
	pbReq := &builderv1.ListBuildsRequest{
		TenantId:    req.TenantID,
		StateFilter: req.State,
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
	}

	resp, err := c.builderService.ListBuilds(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}

	return &ListBuildsResponse{
		Builds:        resp.Msg.Builds,
		NextPageToken: resp.Msg.NextPageToken,
		TotalCount:    resp.Msg.TotalCount,
	}, nil
}

// CancelBuild cancels a running build
func (c *Client) CancelBuild(ctx context.Context, req *CancelBuildRequest) (*CancelBuildResponse, error) {
	pbReq := &builderv1.CancelBuildRequest{
		BuildId:  req.BuildID,
		TenantId: req.TenantID,
	}

	resp, err := c.builderService.CancelBuild(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to cancel build: %w", err)
	}

	return &CancelBuildResponse{
		Success: resp.Msg.Success,
		State:   resp.Msg.State,
	}, nil
}

// DeleteBuild deletes a build and its artifacts
func (c *Client) DeleteBuild(ctx context.Context, req *DeleteBuildRequest) (*DeleteBuildResponse, error) {
	pbReq := &builderv1.DeleteBuildRequest{
		BuildId:  req.BuildID,
		TenantId: req.TenantID,
		Force:    req.Force,
	}

	resp, err := c.builderService.DeleteBuild(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to delete build: %w", err)
	}

	return &DeleteBuildResponse{
		Success: resp.Msg.Success,
	}, nil
}

// StreamBuildLogs streams build logs in real-time
func (c *Client) StreamBuildLogs(ctx context.Context, req *StreamBuildLogsRequest) (*connect.ServerStreamForClient[builderv1.StreamBuildLogsResponse], error) {
	pbReq := &builderv1.StreamBuildLogsRequest{
		BuildId:  req.BuildID,
		TenantId: req.TenantID,
		Follow:   req.Follow,
	}

	stream, err := c.builderService.StreamBuildLogs(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to stream build logs: %w", err)
	}

	return stream, nil
}

// GetTenantQuotas retrieves tenant quotas and usage
func (c *Client) GetTenantQuotas(ctx context.Context, req *GetTenantQuotasRequest) (*GetTenantQuotasResponse, error) {
	pbReq := &builderv1.GetTenantQuotasRequest{
		TenantId: req.TenantID,
	}

	resp, err := c.builderService.GetTenantQuotas(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant quotas: %w", err)
	}

	return &GetTenantQuotasResponse{
		Quotas:     resp.Msg.CurrentLimits,
		Usage:      resp.Msg.CurrentUsage,
		Violations: resp.Msg.Violations,
	}, nil
}

// GetBuildStats retrieves build statistics
func (c *Client) GetBuildStats(ctx context.Context, req *GetBuildStatsRequest) (*GetBuildStatsResponse, error) {
	pbReq := &builderv1.GetBuildStatsRequest{
		TenantId:  req.TenantID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	resp, err := c.builderService.GetBuildStats(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to get build stats: %w", err)
	}

	return &GetBuildStatsResponse{
		Stats: resp.Msg,
	}, nil
}

// GetTenantID returns the tenant ID associated with this client
func (c *Client) GetTenantID() string {
	return c.tenantID
}

// GetServerAddress returns the server address this client is connected to
func (c *Client) GetServerAddress() string {
	return c.serverAddr
}

// tenantTransport adds authentication and tenant isolation headers to all requests
type tenantTransport struct {
	Base     http.RoundTripper
	UserID   string
	TenantID string
}

func (t *tenantTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())
	if req2.Header == nil {
		req2.Header = make(http.Header)
	}

	// Set Authorization header with development token format
	// AIDEV-BUSINESS_RULE: In development, use "dev_user_<id>" format
	// TODO: Update to proper JWT tokens in production
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer dev_user_%s", t.UserID))

	// Also set X-Tenant-ID header for tenant identification
	req2.Header.Set("X-Tenant-ID", t.TenantID)

	// Use the base transport, or default if nil
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}
