package client

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/builderd/v1/builderdv1connect"
)

// AIDEV-NOTE: Builderd client with SPIFFE/SPIRE socket integration
// This client provides a high-level interface for builderd operations with proper authentication

// Config holds the configuration for the builderd client
type Config struct {
	// ServerAddress is the builderd server endpoint (e.g., "https://builderd:8082")
	ServerAddress string

	// I do not understand what this userID is.
	// Is it an unkey user? or is it a linux user on the metal host?
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

type Client struct {
	builderService builderdv1connect.BuilderServiceClient
	tlsProvider    tls.Provider
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

	// Create ConnectRPC client
	builderService := builderdv1connect.NewBuilderServiceClient(
		httpClient,
		config.ServerAddress,
	)

	return &Client{
		builderService: builderService,
		tlsProvider:    tlsProvider,
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
		BuildId: req.BuildID,
	}

	resp, err := c.builderService.GetBuild(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to get build: %w", err)
	}

	return &GetBuildResponse{
		Build: resp.Msg.Build,
	}, nil
}

// ListBuilds lists builds with filtering
func (c *Client) ListBuilds(ctx context.Context, req *ListBuildsRequest) (*ListBuildsResponse, error) {
	pbReq := &builderv1.ListBuildsRequest{
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
		BuildId: req.BuildID,
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
		BuildId: req.BuildID,
		Force:   req.Force,
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
		BuildId: req.BuildID,
		Follow:  req.Follow,
	}

	stream, err := c.builderService.StreamBuildLogs(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to stream build logs: %w", err)
	}

	return stream, nil
}

// GetBuildStats retrieves build statistics
func (c *Client) GetBuildStats(ctx context.Context, req *GetBuildStatsRequest) (*GetBuildStatsResponse, error) {
	pbReq := &builderv1.GetBuildStatsRequest{
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

// GetServerAddress returns the server address this client is connected to
func (c *Client) GetServerAddress() string {
	return c.serverAddr
}
