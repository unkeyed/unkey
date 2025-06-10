package firecracker

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

	"vmm-controlplane/internal/backend/types"
	vmmv1 "vmm-controlplane/gen/vmm/v1"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client implements the Backend interface for Firecracker
type Client struct {
	endpoint   string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Firecracker backend client
func NewClient(endpoint string, logger *slog.Logger) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: createHTTPClient(endpoint),
		logger:     logger.With("backend", "firecracker"),
	}
}

// createHTTPClient creates an HTTP client configured for Unix socket communication
func createHTTPClient(endpoint string) *http.Client {
	socketPath := strings.TrimPrefix(endpoint, "unix://")

	// Create base transport with Unix socket dialer
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}

	// Wrap with OpenTelemetry instrumentation
	instrumentedTransport := otelhttp.NewTransport(transport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("Firecracker %s %s", r.Method, r.URL.Path)
		}),
	)

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: instrumentedTransport,
	}
}

// CreateVM creates a new VM instance with Firecracker
// AIDEV-NOTE: Firecracker requires pre-boot configuration before starting
func (c *Client) CreateVM(ctx context.Context, config *vmmv1.VmConfig) (string, error) {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "creating firecracker vm",
		slog.Int("vcpus", int(config.Cpu.VcpuCount)),
		slog.Int64("memory_bytes", config.Memory.SizeBytes),
	)

	// Convert generic config to Firecracker-specific configs
	machineConfig, bootSource, drives, netInterfaces := c.genericToFirecrackerConfig(config)

	// Step 1: Configure machine (CPU/Memory)
	if err := c.configureMachine(ctx, machineConfig); err != nil {
		return "", fmt.Errorf("failed to configure machine: %w", err)
	}

	// Step 2: Configure boot source (kernel)
	if err := c.configureBootSource(ctx, bootSource); err != nil {
		return "", fmt.Errorf("failed to configure boot source: %w", err)
	}

	// Step 3: Configure drives
	for _, drive := range drives {
		if err := c.configureDrive(ctx, drive); err != nil {
			return "", fmt.Errorf("failed to configure drive %s: %w", drive.DriveID, err)
		}
	}

	// Step 4: Configure network interfaces
	for _, netIface := range netInterfaces {
		if err := c.configureNetworkInterface(ctx, netIface); err != nil {
			return "", fmt.Errorf("failed to configure network interface %s: %w", netIface.IfaceID, err)
		}
	}

	// AIDEV-NOTE: Firecracker doesn't return a VM ID, using a generated one
	vmID := fmt.Sprintf("firecracker-vm-%d", time.Now().Unix())

	c.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker vm configured successfully",
		slog.String("vm_id", vmID),
	)

	return vmID, nil
}

// configureMachine sets the VM's CPU and memory configuration
func (c *Client) configureMachine(ctx context.Context, config firecrackerMachineConfig) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal machine config: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", "/machine-config", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("machine config failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// configureBootSource sets the VM's boot configuration
func (c *Client) configureBootSource(ctx context.Context, config firecrackerBootSource) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal boot source config: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", "/boot-source", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("boot source config failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// configureDrive adds a drive to the VM
func (c *Client) configureDrive(ctx context.Context, drive firecrackerDrive) error {
	body, err := json.Marshal(drive)
	if err != nil {
		return fmt.Errorf("failed to marshal drive config: %w", err)
	}

	path := fmt.Sprintf("/drives/%s", drive.DriveID)
	resp, err := c.doRequest(ctx, "PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("drive config failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// configureNetworkInterface adds a network interface to the VM
func (c *Client) configureNetworkInterface(ctx context.Context, netIface firecrackerNetworkInterface) error {
	body, err := json.Marshal(netIface)
	if err != nil {
		return fmt.Errorf("failed to marshal network interface config: %w", err)
	}

	path := fmt.Sprintf("/network-interfaces/%s", netIface.IfaceID)
	resp, err := c.doRequest(ctx, "PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("network interface config failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// BootVM starts a created VM
func (c *Client) BootVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "booting firecracker vm",
		slog.String("vm_id", vmID),
	)

	action := firecrackerAction{ActionType: "InstanceStart"}
	body, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal boot action: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", "/actions", body)
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

	c.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker vm booted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ShutdownVM gracefully stops a running VM
// AIDEV-NOTE: Firecracker doesn't have graceful shutdown, using power off
func (c *Client) ShutdownVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down firecracker vm",
		slog.String("vm_id", vmID),
	)

	// AIDEV-TODO: Firecracker may need guest-initiated shutdown for graceful stop
	return c.powerOffVM(ctx, vmID)
}

// powerOffVM forcefully stops the VM
func (c *Client) powerOffVM(ctx context.Context, vmID string) error {
	action := firecrackerAction{ActionType: "SendCtrlAltDel"}
	body, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal shutdown action: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", "/actions", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vm shutdown failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PauseVM pauses a running VM
func (c *Client) PauseVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "pausing firecracker vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PATCH", "/vm", []byte(`{"state": "Paused"}`))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vm pause failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker vm paused successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ResumeVM resumes a paused VM
func (c *Client) ResumeVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "resuming firecracker vm",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "PATCH", "/vm", []byte(`{"state": "Resumed"}`))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vm resume failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker vm resumed successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// RebootVM restarts a running VM
// AIDEV-NOTE: Firecracker doesn't have direct reboot, using Ctrl+Alt+Del
func (c *Client) RebootVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "rebooting firecracker vm",
		slog.String("vm_id", vmID),
	)

	action := firecrackerAction{ActionType: "SendCtrlAltDel"}
	body, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal reboot action: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", "/actions", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vm reboot failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker vm reboot signal sent successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// DeleteVM removes a VM instance
// AIDEV-NOTE: Firecracker VMs are deleted when the process terminates
func (c *Client) DeleteVM(ctx context.Context, vmID string) error {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "deleting firecracker vm",
		slog.String("vm_id", vmID),
	)

	// For Firecracker, we need to stop the VM process to delete it
	// This implementation assumes external process management
	c.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker vm deletion requires process termination",
		slog.String("vm_id", vmID),
	)

	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (c *Client) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	c.logger.LogAttrs(ctx, slog.LevelInfo, "getting firecracker vm info",
		slog.String("vm_id", vmID),
	)

	resp, err := c.doRequest(ctx, "GET", "/", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get vm info with status %d: %s", resp.StatusCode, string(respBody))
	}

	var vmInfo firecrackerVMInfo
	if err := json.NewDecoder(resp.Body).Decode(&vmInfo); err != nil {
		return nil, fmt.Errorf("failed to decode vm info: %w", err)
	}

	// Convert Firecracker state to generic state
	state := c.firecrackerStateToGeneric(vmInfo.State)

	c.logger.LogAttrs(ctx, slog.LevelInfo, "retrieved firecracker vm info successfully",
		slog.String("vm_id", vmID),
		slog.String("state", state.String()),
	)

	// AIDEV-TODO: Reconstruct generic config from Firecracker VM info
	return &types.VMInfo{
		State:  state,
		Config: nil, // Config reconstruction would require storing original config
	}, nil
}

// Ping checks if the Firecracker backend is healthy and responsive
func (c *Client) Ping(ctx context.Context) error {
	c.logger.LogAttrs(ctx, slog.LevelDebug, "pinging firecracker backend")

	resp, err := c.doRequest(ctx, "GET", "/", nil)
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

	c.logger.LogAttrs(ctx, slog.LevelDebug, "firecracker ping successful")
	return nil
}

// doRequest performs an HTTP request to the Firecracker API
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

	c.logger.LogAttrs(ctx, slog.LevelDebug, "sending firecracker request",
		slog.String("method", method),
		slog.String("url", url),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "firecracker request failed",
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

// Ensure Client implements Backend interface
var _ types.Backend = (*Client)(nil)
