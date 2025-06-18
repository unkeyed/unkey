package cloudhypervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client implements the Backend interface for Cloud Hypervisor
type Client struct {
	endpoint   string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Cloud Hypervisor backend client
func NewClient(endpoint string, logger *slog.Logger) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: createHTTPClient(endpoint),
		logger:     logger.With("backend", "cloudhypervisor"),
	}
}

// createHTTPClient creates an HTTP client configured for Unix socket communication
func createHTTPClient(endpoint string) *http.Client {
	socketPath := strings.TrimPrefix(endpoint, "unix://")

	// Create base transport with Unix socket dialer
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath) //nolint:exhaustruct
		},
	}

	// Wrap with OpenTelemetry instrumentation
	instrumentedTransport := otelhttp.NewTransport(transport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("CloudHypervisor %s %s", r.Method, r.URL.Path)
		}),
	)

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: instrumentedTransport,
	}
}

// CreateVM creates a new VM instance
func (c *Client) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "creating vm",
		slog.Int("vcpus", int(config.GetCpu().GetVcpuCount())),
		slog.Int64("memory_bytes", config.GetMemory().GetSizeBytes()),
	)

	// Convert generic config to Cloud Hypervisor API format
	chConfig := c.genericToCloudHypervisorConfig(config)

	body, err := json.Marshal(chConfig)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "failed to marshal vm config",
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.create", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm creation failed",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return "", fmt.Errorf("vm creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// AIDEV-NOTE: Cloud Hypervisor doesn't return a VM ID, using a generated one
	vmID := fmt.Sprintf("vm-%d", time.Now().Unix())

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm created successfully",
		slog.String("vm_id", vmID),
	)

	return vmID, nil
}

// DeleteVM removes a VM instance
func (c *Client) DeleteVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "deleting vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.delete", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm deletion failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm deletion failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm deleted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// BootVM starts a created VM
func (c *Client) BootVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "booting vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.boot", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm boot failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm boot failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm booted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ShutdownVM gracefully stops a running VM
func (c *Client) ShutdownVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.shutdown", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm shutdown failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm shutdown failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm shutdown successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ShutdownVMWithOptions gracefully stops a running VM with force and timeout options
func (c *Client) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down vm with options",
		slog.String("vm_id", vmID),
		slog.Bool("force", force),
		slog.Int("timeout_seconds", int(timeoutSeconds)),
	)

	// AIDEV-NOTE: Cloud Hypervisor API doesn't currently support shutdown options
	// For now, delegate to regular shutdown regardless of force/timeout flags
	if force {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "force shutdown requested, using standard shutdown",
			slog.String("vm_id", vmID),
		)
	}

	// Use standard shutdown endpoint - Cloud Hypervisor handles graceful shutdown internally
	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.shutdown", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm shutdown with options failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm shutdown failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm shutdown with options completed successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// PauseVM pauses a running VM
func (c *Client) PauseVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "pausing vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.pause", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm pause failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm pause failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm paused successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ResumeVM resumes a paused VM
func (c *Client) ResumeVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "resuming vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.resume", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm resume failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm resume failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm resumed successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// RebootVM restarts a running VM
func (c *Client) RebootVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "rebooting vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PUT", "/api/v1/vm.reboot", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "vm reboot failed",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("vm reboot failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "vm rebooted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (c *Client) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "getting vm info",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "GET", "/api/v1/vm.info", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "failed to get vm info",
			slog.String("vm_id", vmID),
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return nil, fmt.Errorf("failed to get vm info with status %d: %s", resp.StatusCode, string(respBody))
	}

	var vmInfo cloudHypervisorVMInfo
	if err := json.NewDecoder(resp.Body).Decode(&vmInfo); err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "failed to decode vm info",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to decode vm info: %w", err)
	}

	// Convert Cloud Hypervisor state to generic state
	state := c.cloudHypervisorStateToGeneric(vmInfo.State)

	c.logger.LogAttrs(ctx, slog.LevelInfo, "retrieved vm info successfully",
		slog.String("vm_id", vmID),
		slog.String("state", state.String()),
	)

	// AIDEV-TODO: Implement config reconstruction from VM info
	//exhaustruct:ignore
	return &types.VMInfo{
		State:  state,
		Config: nil, // Config reconstruction would require storing original config
	}, nil
}

// doRequest performs an HTTP request to the Cloud Hypervisor API
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	url := c.buildURL(path)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "failed to create request",
			slog.String("method", method),
			slog.String("path", path),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "sending request",
		slog.String("method", method),
		slog.String("url", url),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "request failed",
			slog.String("method", method),
			slog.String("url", url),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// buildURL constructs the full URL for the API request
func (c *Client) buildURL(path string) string {
	// AIDEV-NOTE: For Unix sockets, use http://localhost as the URL
	// The actual socket connection is handled by the custom transport
	return "http://localhost" + path
}

// Ping checks if the Cloud Hypervisor backend is healthy and responsive
func (c *Client) Ping(ctx context.Context) error {
	c.logger.LogAttrs(ctx, slog.LevelDebug, "pinging cloud hypervisor backend")

	resp, err := c.doRequest(ctx, "GET", "/api/v1/vmm.ping", nil)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "ping request failed",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("ping request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.LogAttrs(ctx, slog.LevelError, "ping failed",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return fmt.Errorf("ping failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "ping successful")
	return nil
}

// GetVMMetrics retrieves current VM resource usage metrics from Cloud Hypervisor
func (c *Client) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	c.logger.LogAttrs(ctx, slog.LevelDebug, "getting cloud hypervisor vm metrics",
		slog.String("vm_id", vmID),
	)

	// TODO: Implement Cloud Hypervisor metrics collection
	// For now, return stub data to satisfy the interface
	return &types.VMMetrics{
		Timestamp:        time.Now(),
		CpuTimeNanos:     0,
		MemoryUsageBytes: 0,
		DiskReadBytes:    0,
		DiskWriteBytes:   0,
		NetworkRxBytes:   0,
		NetworkTxBytes:   0,
	}, nil
}

// Ensure Client implements Backend interface
var _ types.Backend = (*Client)(nil)
