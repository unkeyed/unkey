package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1/vmprovisionerv1connect"
)

// This client provides a high-level interface for metald VM operations with proper authentication

// Config holds the configuration for the metald client
type Config struct {
	// ServerAddress is the metald server endpoint (e.g., "https://metald:8080")
	ServerAddress string

	// UserID is the user identifier for authentication
	UserID string

	// TenantID is the tenant identifier for data scoping
	TenantID string

	// ProjectID identifies the tenants project
	ProjectID string

	// EnvironmentID identifies the environment within the project
	EnvironmentID string

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
	vmService     vmprovisionerv1connect.VmServiceClient
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
		Base:          httpClient.Transport,
		ProjectID:     config.ProjectID,
		TenantID:      config.TenantID,
		EnvironmentID: config.EnvironmentID,
	}

	// Create ConnectRPC client
	vmService := vmprovisionerv1connect.NewVmServiceClient(
		httpClient,
		config.ServerAddress,
	)

	return &Client{
		vmService:     vmService,
		tlsProvider:   tlsProvider,
		tenantID:      config.TenantID,
		projectID:     config.ProjectID,
		environmentID: config.EnvironmentID,
		serverAddr:    config.ServerAddress,
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
func (c *Client) CreateVM(ctx context.Context, req *CreateVMRequest) (*CreateVMResponse, error) {
	// Convert to protobuf request
	pbReq := &vmprovisionerv1.CreateVmRequest{
		VmId:     req.VMID,
		Config:   req.Config,
		TenantId: c.tenantID,
	}

	resp, err := c.vmService.CreateVm(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	return &CreateVMResponse{
		VMID:  resp.Msg.VmId,
		State: resp.Msg.State,
	}, nil
}

// BootVM starts a created virtual machine
func (c *Client) BootVM(ctx context.Context, vmID string) (*BootVMResponse, error) {
	req := &vmprovisionerv1.BootVmRequest{
		VmId: vmID,
	}

	resp, err := c.vmService.BootVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to boot VM: %w", err)
	}

	return &BootVMResponse{
		Success: resp.Msg.Success,
		State:   resp.Msg.State,
	}, nil
}

// ShutdownVM gracefully stops a running virtual machine
func (c *Client) ShutdownVM(ctx context.Context, req *ShutdownVMRequest) (*ShutdownVMResponse, error) {
	pbReq := &vmprovisionerv1.ShutdownVmRequest{
		VmId:           req.VMID,
		Force:          req.Force,
		TimeoutSeconds: int32(req.TimeoutSeconds),
	}

	resp, err := c.vmService.ShutdownVm(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to shutdown VM: %w", err)
	}

	return &ShutdownVMResponse{
		Success: resp.Msg.Success,
		State:   resp.Msg.State,
	}, nil
}

// DeleteVM removes a virtual machine
func (c *Client) DeleteVM(ctx context.Context, req *DeleteVMRequest) (*DeleteVMResponse, error) {
	pbReq := &vmprovisionerv1.DeleteVmRequest{
		VmId:  req.VMID,
		Force: req.Force,
	}

	resp, err := c.vmService.DeleteVm(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to delete VM: %w", err)
	}

	return &DeleteVMResponse{
		Success: resp.Msg.Success,
	}, nil
}

// GetVMInfo retrieves detailed information about a virtual machine
func (c *Client) GetVMInfo(ctx context.Context, vmID string) (*VMInfo, error) {
	req := &vmprovisionerv1.GetVmInfoRequest{
		VmId: vmID,
	}

	resp, err := c.vmService.GetVmInfo(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	return &VMInfo{
		VMID:        resp.Msg.VmId,
		State:       resp.Msg.State,
		Config:      resp.Msg.Config,
		Metrics:     resp.Msg.Metrics,
		NetworkInfo: resp.Msg.NetworkInfo,
	}, nil
}

// ListVMs retrieves a list of virtual machines for the authenticated customer
func (c *Client) ListVMs(ctx context.Context, req *ListVMsRequest) (*ListVMsResponse, error) {
	pbReq := &vmprovisionerv1.ListVmsRequest{
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}

	resp, err := c.vmService.ListVms(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	return &ListVMsResponse{
		VMs:           resp.Msg.Vms,
		NextPageToken: resp.Msg.NextPageToken,
		TotalCount:    resp.Msg.TotalCount,
	}, nil
}

// PauseVM pauses a running virtual machine
func (c *Client) PauseVM(ctx context.Context, vmID string) (*PauseVMResponse, error) {
	req := &vmprovisionerv1.PauseVmRequest{
		VmId: vmID,
	}

	resp, err := c.vmService.PauseVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to pause VM: %w", err)
	}

	return &PauseVMResponse{
		Success: resp.Msg.Success,
		State:   resp.Msg.State,
	}, nil
}

// ResumeVM resumes a paused virtual machine
func (c *Client) ResumeVM(ctx context.Context, vmID string) (*ResumeVMResponse, error) {
	req := &vmprovisionerv1.ResumeVmRequest{
		VmId: vmID,
	}

	resp, err := c.vmService.ResumeVm(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to resume VM: %w", err)
	}

	return &ResumeVMResponse{
		Success: resp.Msg.Success,
		State:   resp.Msg.State,
	}, nil
}

// RebootVM restarts a virtual machine
func (c *Client) RebootVM(ctx context.Context, req *RebootVMRequest) (*RebootVMResponse, error) {
	pbReq := &vmprovisionerv1.RebootVmRequest{
		VmId:  req.VMID,
		Force: req.Force,
	}

	resp, err := c.vmService.RebootVm(ctx, connect.NewRequest(pbReq))
	if err != nil {
		return nil, fmt.Errorf("failed to reboot VM: %w", err)
	}

	return &RebootVMResponse{
		Success: resp.Msg.Success,
		State:   resp.Msg.State,
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
