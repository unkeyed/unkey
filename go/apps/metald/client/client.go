package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"
)

// This client provides a high-level interface for metald VM operations with proper authentication

// Config holds the configuration for the metald client
type Config struct {
	// ServerAddress is the metald server endpoint (e.g., "https://metald:8080")
	ServerAddress string

	// UserID is the user identifier for authentication
	UserID string

	DeploymentID string

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

// Client provides a high-level interface to metald services
type Client struct {
	vmService     metaldv1connect.VmServiceClient
	tlsProvider   tls.Provider
	tenantID      string
	projectID     string
	environmentID string
	serverAddr    string
}

// New creates a new metald client with SPIFFE/SPIRE integration
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
		Base: httpClient.Transport,
	}

	// Create ConnectRPC client
	vmService := metaldv1connect.NewVmServiceClient(
		httpClient,
		config.ServerAddress,
	)

	return &Client{
		vmService:   vmService,
		tlsProvider: tlsProvider,
		serverAddr:  config.ServerAddress,
	}, nil
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	if c.tlsProvider != nil {
		return c.tlsProvider.Close()
	}
	return nil
}

// CreateVM creates a new virtual machine with the specified configuration
func (c *Client) CreateVM(ctx context.Context, req *metaldv1.CreateVmRequest) (*metaldv1.CreateVmResponse, error) {
	resp, err := c.vmService.CreateVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}
	return resp.Msg, nil
}

// BootVM starts a created virtual machine
func (c *Client) BootVM(ctx context.Context, req *metaldv1.BootVmRequest) (*metaldv1.BootVmResponse, error) {

	resp, err := c.vmService.BootVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to boot VM: %w", err)
	}
	return resp.Msg, nil
}

// ShutdownVM gracefully stops a running virtual machine
func (c *Client) ShutdownVM(ctx context.Context, req *metaldv1.ShutdownVmRequest) (*metaldv1.ShutdownVmResponse, error) {
	resp, err := c.vmService.ShutdownVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to shutdown VM: %w", err)
	}
	return resp.Msg, nil
}

// DeleteVM removes a virtual machine
func (c *Client) DeleteVM(ctx context.Context, req *metaldv1.DeleteVmRequest) (*metaldv1.DeleteVmResponse, error) {
	resp, err := c.vmService.DeleteVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to delete VM: %w", err)
	}
	return resp.Msg, nil
}

// GetVMInfo retrieves detailed information about a virtual machine
func (c *Client) GetVMInfo(ctx context.Context, req *metaldv1.GetVmInfoRequest) (*metaldv1.GetVmInfoResponse, error) {
	resp, err := c.vmService.GetVmInfo(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}
	return resp.Msg, nil
}

// ListVMs retrieves a list of virtual machines for the authenticated customer
func (c *Client) ListVMs(ctx context.Context, req *metaldv1.ListVmsRequest) (*metaldv1.ListVmsResponse, error) {
	resp, err := c.vmService.ListVms(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}
	return resp.Msg, nil
}

// PauseVM pauses a running virtual machine
func (c *Client) PauseVM(ctx context.Context, req *metaldv1.PauseVmRequest) (*metaldv1.PauseVmResponse, error) {
	resp, err := c.vmService.PauseVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to pause VM: %w", err)
	}
	return resp.Msg, nil
}

// ResumeVM resumes a paused virtual machine
func (c *Client) ResumeVM(ctx context.Context, req *metaldv1.ResumeVmRequest) (*metaldv1.ResumeVmResponse, error) {
	resp, err := c.vmService.ResumeVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to resume VM: %w", err)
	}
	return resp.Msg, nil
}

// RebootVM restarts a virtual machine
func (c *Client) RebootVM(ctx context.Context, req *metaldv1.RebootVmRequest) (*metaldv1.RebootVmResponse, error) {
	resp, err := c.vmService.RebootVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to reboot VM: %w", err)
	}
	return resp.Msg, nil
}

// GetTenantID returns the tenant ID associated with this client
func (c *Client) GetTenantID() string {
	return c.tenantID
}

// GetServerAddress returns the server address this client is connected to
func (c *Client) GetServerAddress() string {
	return c.serverAddr
}

// CreateDeployment creates a new deployment with multiple VMs
func (c *Client) CreateDeployment(ctx context.Context, req *metaldv1.CreateDeploymentRequest) (*metaldv1.CreateDeploymentResponse, error) {
	resp, err := c.vmService.CreateDeployment(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}
	return resp.Msg, nil
}

// UpdateDeployment updates an existing deployment
func (c *Client) UpdateDeployment(ctx context.Context, req *metaldv1.UpdateDeploymentRequest) (*metaldv1.UpdateDeploymentResponse, error) {
	resp, err := c.vmService.UpdateDeployment(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to update deployment: %w", err)
	}
	return resp.Msg, nil
}

// DeleteDeployment deletes an existing deployment
func (c *Client) DeleteDeployment(ctx context.Context, req *metaldv1.DeleteDeploymentRequest) (*metaldv1.DeleteDeploymentResponse, error) {
	resp, err := c.vmService.DeleteDeployment(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to delete deployment: %w", err)
	}
	return resp.Msg, nil
}

// GetDeployment retrieves information about a deployment
func (c *Client) GetDeployment(ctx context.Context, req *metaldv1.GetDeploymentRequest) (*metaldv1.GetDeploymentResponse, error) {
	resp, err := c.vmService.GetDeployment(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	return resp.Msg, nil
}

// tenantTransport adds authentication and tenant isolation headers to all requests
type tenantTransport struct {
	Base          http.RoundTripper
	EnvironmentID string
	ProjectID     string
	TenantID      string
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
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer dev_user_%s", t.TenantID))

	// Also set X-Tenant-ID header for tenant identification
	req2.Header.Set("X-Tenant-ID", t.TenantID)
	req2.Header.Set("X-Project-ID", t.ProjectID)
	req2.Header.Set("X-Environment-ID", t.EnvironmentID)

	// Use the base transport, or default if nil
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}
