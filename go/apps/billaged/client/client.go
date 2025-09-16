package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	billingv1 "github.com/unkeyed/unkey/go/gen/proto/billaged/v1"
	"github.com/unkeyed/unkey/go/gen/proto/billaged/v1/billagedv1connect"
)

// AIDEV-NOTE: Billaged client with SPIFFE/SPIRE socket integration
// This client provides a high-level interface for billaged operations with proper authentication

// Config holds the configuration for the billaged client
type Config struct {
	// ServerAddress is the billaged server endpoint (e.g., "https://billaged:8081")
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

// Client provides a high-level interface to billaged services
type Client struct {
	billingService billagedv1connect.BillingServiceClient
	tlsProvider    tls.Provider
	userID         string
	tenantID       string
	serverAddr     string
}

// New creates a new billaged client with SPIFFE/SPIRE integration
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
	billingService := billagedv1connect.NewBillingServiceClient(
		httpClient,
		config.ServerAddress,
	)

	return &Client{
		billingService: billingService,
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

// SendMetricsBatch sends a batch of VM metrics to the billing service
func (c *Client) SendMetricsBatch(ctx context.Context, req *SendMetricsBatchRequest) (*SendMetricsBatchResponse, error) {
	pbReq := &billingv1.SendMetricsBatchRequest{
		VmId:       req.VmID,
		CustomerId: req.CustomerID,
		Metrics:    req.Metrics,
	}

	resp, err := c.billingService.SendMetricsBatch(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to send metrics batch: %w", err)
	}

	return &SendMetricsBatchResponse{
		Success: resp.Msg.Success,
		Message: resp.Msg.Message,
	}, nil
}

// SendHeartbeat sends a heartbeat to indicate this instance is alive
func (c *Client) SendHeartbeat(ctx context.Context, req *SendHeartbeatRequest) (*SendHeartbeatResponse, error) {
	pbReq := &billingv1.SendHeartbeatRequest{
		InstanceId: req.InstanceID,
		ActiveVms:  req.ActiveVMs,
	}

	resp, err := c.billingService.SendHeartbeat(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to send heartbeat: %w", err)
	}

	return &SendHeartbeatResponse{
		Success: resp.Msg.Success,
	}, nil
}

// NotifyVmStarted notifies the billing service that a VM has started
func (c *Client) NotifyVmStarted(ctx context.Context, req *NotifyVmStartedRequest) (*NotifyVmStartedResponse, error) {
	pbReq := &billingv1.NotifyVmStartedRequest{
		VmId:       req.VmID,
		CustomerId: req.CustomerID,
		StartTime:  req.StartTime,
	}

	resp, err := c.billingService.NotifyVmStarted(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to notify VM started: %w", err)
	}

	return &NotifyVmStartedResponse{
		Success: resp.Msg.Success,
	}, nil
}

// NotifyVmStopped notifies the billing service that a VM has stopped
func (c *Client) NotifyVmStopped(ctx context.Context, req *NotifyVmStoppedRequest) (*NotifyVmStoppedResponse, error) {
	pbReq := &billingv1.NotifyVmStoppedRequest{
		VmId:     req.VmID,
		StopTime: req.StopTime,
	}

	resp, err := c.billingService.NotifyVmStopped(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to notify VM stopped: %w", err)
	}

	return &NotifyVmStoppedResponse{
		Success: resp.Msg.Success,
	}, nil
}

// NotifyPossibleGap notifies about a possible gap in metrics reporting
func (c *Client) NotifyPossibleGap(ctx context.Context, req *NotifyPossibleGapRequest) (*NotifyPossibleGapResponse, error) {
	pbReq := &billingv1.NotifyPossibleGapRequest{
		VmId:       req.VmID,
		LastSent:   req.LastSent,
		ResumeTime: req.ResumeTime,
	}

	resp, err := c.billingService.NotifyPossibleGap(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to notify possible gap: %w", err)
	}

	return &NotifyPossibleGapResponse{
		Success: resp.Msg.Success,
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
