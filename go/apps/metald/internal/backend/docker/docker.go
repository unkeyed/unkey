package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Compile-time interface checks
var _ types.Backend = (*Backend)(nil)
var _ types.VMListProvider = (*Backend)(nil)

// Backend implements types.Backend using Docker containers
type Backend struct {
	logger          logging.Logger
	dockerClient    *client.Client
	isRunningDocker bool // Whether this service is running in Docker
}

// New creates a new Docker backend
func New(logger logging.Logger, isRunningDocker bool) (*Backend, error) {
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify Docker connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, pingErr := dockerClient.Ping(ctx); pingErr != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", pingErr)
	}

	return &Backend{
		logger:          logger.With("backend", "docker"),
		dockerClient:    dockerClient,
		isRunningDocker: isRunningDocker,
	}, nil
}

// calculateGatewayIP calculates the gateway IP for a subnet (usually .1)
func calculateGatewayIP(subnet string) (string, error) {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet: %w", err)
	}

	// Get the first IP in the subnet (network address)
	ip := ipnet.IP.Mask(ipnet.Mask)

	// Increment to get the first usable IP (gateway)
	ip[len(ip)-1]++

	return ip.String(), nil
}

// createCustomNetwork creates a Docker network for the deployment
func (b *Backend) createCustomNetwork(ctx context.Context, deploymentID, subnet string) (string, error) {
	networkName := fmt.Sprintf("unkey-deployment-%s", deploymentID)

	// Check if network already exists
	networks, err := b.dockerClient.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	if len(networks) > 0 {
		b.logger.Info("network already exists", "network", networkName)
		return networks[0].ID, nil
	}

	// Calculate gateway IP
	gatewayIP, err := calculateGatewayIP(subnet)
	if err != nil {
		return "", fmt.Errorf("failed to calculate gateway IP: %w", err)
	}

	// Create new network
	networkConfig := network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  subnet,
					Gateway: gatewayIP,
				},
			},
		},
		Options: map[string]string{
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.bridge.enable_icc":           "true",
		},
		Labels: map[string]string{
			"unkey.managed.by":    "metald",
			"unkey.deployment.id": deploymentID,
		},
	}

	resp, err := b.dockerClient.NetworkCreate(ctx, networkName, networkConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	b.logger.Info("created custom network",
		"network", networkName,
		"subnet", subnet,
		"gateway", gatewayIP)

	return resp.ID, nil
}

// CreateVM creates a new Docker container as a VM
func (b *Backend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	// Use the provided VM ID from config
	vmID := config.Id
	if vmID == "" {
		return "", fmt.Errorf("VM ID is required")
	}

	b.logger.Info("creating Docker VM",
		"vm_id", vmID,
		"image", config.Boot)

	// Use boot configuration as Docker image name
	imageName := config.Boot
	if imageName == "" {
		return "", fmt.Errorf("boot image (Docker image) is required")
	}

	// Parse network configuration
	var networkInfo map[string]string
	if config.NetworkConfig != "" {
		if err := json.Unmarshal([]byte(config.NetworkConfig), &networkInfo); err != nil {
			return "", fmt.Errorf("failed to parse network config: %w", err)
		}
	}

	// Pull image if not present
	if err := b.pullImageIfNeeded(ctx, imageName); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Use VM ID directly as container name
	containerName := vmID
	containerConfig := &container.Config{
		Image: imageName,
		Labels: map[string]string{
			"unkey.vm.id":      vmID,
			"unkey.managed.by": "metald",
		},
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
		Hostname: containerName,
	}

	// Set memory and CPU limits based on VM config
	resources := container.Resources{}
	if config.MemorySizeMib > 0 {
		resources.Memory = int64(config.MemorySizeMib) * 1024 * 1024
	}
	if config.VcpuCount > 0 {
		resources.NanoCPUs = int64(config.VcpuCount) * 1000000000
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "0", // Let Docker assign a random port
				},
			},
		},
		AutoRemove: false,
		Resources:  resources,
	}

	var networkingConfig *network.NetworkingConfig

	// If we have network info, create/use custom network
	if deploymentID, ok := networkInfo["deployment_id"]; ok {
		subnet := networkInfo["subnet"]
		allocatedIP := networkInfo["allocated_ip"]

		// Create custom network for this deployment
		_, err := b.createCustomNetwork(ctx, deploymentID, subnet)
		if err != nil {
			return "", fmt.Errorf("failed to create custom network: %w", err)
		}

		networkName := fmt.Sprintf("unkey-deployment-%s", deploymentID)
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkName: {
					IPAMConfig: &network.EndpointIPAMConfig{
						IPv4Address: allocatedIP,
					},
				},
			},
		}

		b.logger.Info("using custom network",
			"vm_id", vmID,
			"network", networkName,
			"ip", allocatedIP)
	}

	// Create the container
	resp, err := b.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	b.logger.Info("Docker VM created",
		"vm_id", vmID,
		"container_id", resp.ID)

	return vmID, nil
}

// DeleteVM removes a Docker container VM
func (b *Backend) DeleteVM(ctx context.Context, vmID string) error {
	// Get container info to find which networks it's connected to
	inspect, err := b.dockerClient.ContainerInspect(ctx, vmID)
	if err != nil && !errdefs.IsNotFound(err) {
		b.logger.Warn("failed to inspect container", "vm_id", vmID, "error", err)
	}

	// Stop container if running
	if err := b.dockerClient.ContainerStop(ctx, vmID, container.StopOptions{}); err != nil {
		if !errdefs.IsNotFound(err) {
			b.logger.Error("failed to stop container",
				"vm_id", vmID,
				"error", err)
		}
	}

	// Remove container
	if err := b.dockerClient.ContainerRemove(ctx, vmID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	// Clean up deployment networks if this was the last container
	if inspect.ID != "" {
		for networkName := range inspect.NetworkSettings.Networks {
			if strings.HasPrefix(networkName, "unkey-deployment-") {
				b.cleanupNetworkIfEmpty(ctx, networkName)
			}
		}
	}

	b.logger.Info("Docker VM deleted", "vm_id", vmID)
	return nil
}

// cleanupNetworkIfEmpty removes a deployment network if no containers are using it
func (b *Backend) cleanupNetworkIfEmpty(ctx context.Context, networkName string) {
	// Get network info
	networkInfo, err := b.dockerClient.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		b.logger.Warn("failed to inspect network", "network", networkName, "error", err)
		return
	}

	// Count VM containers (containers managed by metald)
	vmContainers := 0
	for _, endpoint := range networkInfo.Containers {
		// Count containers that look like VM containers
		if strings.HasPrefix(endpoint.Name, "ud-") || strings.Contains(endpoint.Name, "unkey-vm") {
			vmContainers++
		}
	}

	// If no VM containers are left, remove the network
	if vmContainers == 0 {
		// Remove the network
		err := b.dockerClient.NetworkRemove(ctx, networkName)
		if err != nil {
			b.logger.Warn("failed to remove empty network", "network", networkName, "error", err)
		} else {
			b.logger.Info("cleaned up empty deployment network", "network", networkName)
		}
	}
}

// BootVM starts a Docker container VM
func (b *Backend) BootVM(ctx context.Context, vmID string) error {
	// Docker client accepts both container ID and container name
	if err := b.dockerClient.ContainerStart(ctx, vmID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	b.logger.Info("Docker VM booted", "vm_id", vmID)
	return nil
}

// ShutdownVM gracefully stops a Docker container VM
func (b *Backend) ShutdownVM(ctx context.Context, vmID string) error {
	return b.ShutdownVMWithOptions(ctx, vmID, false, 30)
}

// ShutdownVMWithOptions gracefully stops a Docker container VM with options
func (b *Backend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	timeout := int(timeoutSeconds)
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := b.dockerClient.ContainerStop(ctx, vmID, stopOptions); err != nil {
		if force {
			// Force kill if graceful stop fails
			if killErr := b.dockerClient.ContainerKill(ctx, vmID, "KILL"); killErr != nil {
				return fmt.Errorf("failed to force kill container: %w", killErr)
			}
		} else {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	b.logger.Info("Docker VM shutdown", "vm_id", vmID, "force", force)
	return nil
}

// PauseVM pauses a Docker container VM
func (b *Backend) PauseVM(ctx context.Context, vmID string) error {
	if err := b.dockerClient.ContainerPause(ctx, vmID); err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	b.logger.Info("Docker VM paused", "vm_id", vmID)
	return nil
}

// ResumeVM resumes a paused Docker container VM
func (b *Backend) ResumeVM(ctx context.Context, vmID string) error {
	if err := b.dockerClient.ContainerUnpause(ctx, vmID); err != nil {
		return fmt.Errorf("failed to unpause container: %w", err)
	}

	b.logger.Info("Docker VM resumed", "vm_id", vmID)
	return nil
}

// RebootVM restarts a Docker container VM
func (b *Backend) RebootVM(ctx context.Context, vmID string) error {
	timeout := int(30)
	if err := b.dockerClient.ContainerRestart(ctx, vmID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	b.logger.Info("Docker VM rebooted", "vm_id", vmID)
	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (b *Backend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	// Get current container state
	inspect, err := b.dockerClient.ContainerInspect(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Determine state based on container state
	state := metaldv1.VmState_VM_STATE_UNSPECIFIED
	if inspect.State.Running {
		state = metaldv1.VmState_VM_STATE_RUNNING
	} else if inspect.State.Paused {
		state = metaldv1.VmState_VM_STATE_PAUSED
	} else if inspect.State.Dead || inspect.State.OOMKilled {
		state = metaldv1.VmState_VM_STATE_SHUTDOWN
	} else {
		state = metaldv1.VmState_VM_STATE_CREATED
	}

	// Reconstruct config from container labels and inspect data
	config := &metaldv1.VmConfig{
		Id:   vmID,
		Boot: inspect.Config.Image,
	}

	// Extract resources if set
	if inspect.HostConfig.Resources.Memory > 0 {
		config.MemorySizeMib = uint64(inspect.HostConfig.Resources.Memory / (1024 * 1024))
	}
	if inspect.HostConfig.Resources.NanoCPUs > 0 {
		config.VcpuCount = uint32(inspect.HostConfig.Resources.NanoCPUs / 1000000000)
	}

	return &types.VMInfo{
		Config: config,
		State:  state,
	}, nil
}

// GetVMMetrics retrieves current VM resource usage metrics
func (b *Backend) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	// Get container stats
	stats, err := b.dockerClient.ContainerStats(ctx, vmID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	// Parse stats from JSON stream
	var containerStats container.Stats
	if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Calculate metrics
	metrics := &types.VMMetrics{
		Timestamp:        time.Now(),
		CpuTimeNanos:     int64(containerStats.CPUStats.CPUUsage.TotalUsage),
		MemoryUsageBytes: int64(containerStats.MemoryStats.Usage),
	}

	// Network stats (sum all interfaces)
	for _, network := range containerStats.Networks {
		metrics.NetworkRxBytes += int64(network.RxBytes)
		metrics.NetworkTxBytes += int64(network.TxBytes)
	}

	// Disk I/O stats (if available)
	for _, ioStats := range containerStats.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(ioStats.Op) {
		case "read":
			metrics.DiskReadBytes += int64(ioStats.Value)
		case "write":
			metrics.DiskWriteBytes += int64(ioStats.Value)
		}
	}

	return metrics, nil
}

// Ping checks if the Docker daemon is healthy and responsive
func (b *Backend) Ping(ctx context.Context) error {
	_, err := b.dockerClient.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker daemon ping failed: %w", err)
	}
	return nil
}

// ListVMs returns a list of all VMs managed by this backend
func (b *Backend) ListVMs() []types.ListableVMInfo {
	ctx := context.Background()
	containers, err := b.dockerClient.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "unkey.managed.by=metald"),
		),
	})
	if err != nil {
		b.logger.Error("failed to list containers", "error", err)
		return []types.ListableVMInfo{}
	}

	vms := make([]types.ListableVMInfo, 0, len(containers))
	for _, c := range containers {
		vmID := c.Labels["unkey.vm.id"]
		if vmID == "" {
			// Fallback to container name since we set container name = VM ID
			vmID = c.Names[0]
		}

		// Determine state from container state
		state := metaldv1.VmState_VM_STATE_CREATED
		if c.State == "running" {
			state = metaldv1.VmState_VM_STATE_RUNNING
		} else if c.State == "paused" {
			state = metaldv1.VmState_VM_STATE_PAUSED
		} else if c.State == "exited" || c.State == "dead" {
			state = metaldv1.VmState_VM_STATE_SHUTDOWN
		}

		config := &metaldv1.VmConfig{
			Id:   vmID,
			Boot: c.Image,
		}

		vms = append(vms, types.ListableVMInfo{
			ID:     vmID,
			State:  state,
			Config: config,
		})
	}
	return vms
}

// Helper methods

func (b *Backend) pullImageIfNeeded(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, _, inspectErr := b.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if inspectErr == nil {
		b.logger.Info("image found locally", "image", imageName)
		return nil
	}

	// Only attempt to pull if the error indicates the image is not found
	if !errdefs.IsNotFound(inspectErr) {
		return fmt.Errorf("failed to inspect image %s: %w", imageName, inspectErr)
	}

	// Image not found locally, attempt to pull it
	b.logger.Info("pulling image", "image", imageName)
	reader, pullErr := b.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if pullErr != nil {
		return fmt.Errorf("image %s not found locally and pull failed: inspect error: %v, pull error: %w", imageName, inspectErr, pullErr)
	}
	defer reader.Close()

	// Read the output to ensure pull completes
	_, err := io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	b.logger.Info("image pulled successfully", "image", imageName)
	return nil
}
