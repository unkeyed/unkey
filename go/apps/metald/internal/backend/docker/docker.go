package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Compile-time interface checks
var _ types.Backend = (*Backend)(nil)
var _ types.VMListProvider = (*Backend)(nil)

// Backend implements types.Backend using Docker containers
type Backend struct {
	logger          logging.Logger
	dockerClient    *client.Client
	vms             map[string]*vmInfo
	mutex           sync.RWMutex
	isRunningDocker bool // Whether this service is running in Docker
}

type vmInfo struct {
	vmID        string
	containerID string
	config      *metaldv1.VmConfig
	state       metaldv1.VmState
	createdAt   time.Time
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
		vms:             make(map[string]*vmInfo),
		isRunningDocker: isRunningDocker,
	}, nil
}

// CreateVM creates a new Docker container as a VM
func (b *Backend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	vmID := uid.New("vm")
	b.logger.Info("creating Docker VM",
		"vm_id", vmID,
		"image", config.Boot)

	// Use boot configuration as Docker image name
	imageName := config.Boot
	if imageName == "" {
		return "", fmt.Errorf("boot image (Docker image) is required")
	}

	// Pull image if not present
	if err := b.pullImageIfNeeded(ctx, imageName); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container configuration
	containerName := fmt.Sprintf("unkey-vm-%s", vmID)
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

	// Create the container
	resp, err := b.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Store VM info
	b.mutex.Lock()
	b.vms[vmID] = &vmInfo{
		vmID:        vmID,
		containerID: resp.ID,
		config:      config,
		state:       metaldv1.VmState_VM_STATE_CREATED,
		createdAt:   time.Now(),
	}
	b.mutex.Unlock()

	b.logger.Info("Docker VM created",
		"vm_id", vmID,
		"container_id", resp.ID)

	return vmID, nil
}

// DeleteVM removes a Docker container VM
func (b *Backend) DeleteVM(ctx context.Context, vmID string) error {
	b.mutex.Lock()
	vm, exists := b.vms[vmID]
	if exists {
		delete(b.vms, vmID)
	}
	b.mutex.Unlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// Stop container if running
	if err := b.dockerClient.ContainerStop(ctx, vm.containerID, container.StopOptions{}); err != nil {
		if !errdefs.IsNotFound(err) {
			b.logger.Error("failed to stop container",
				"vm_id", vmID,
				"container_id", vm.containerID,
				"error", err)
		}
	}

	// Remove container
	if err := b.dockerClient.ContainerRemove(ctx, vm.containerID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	b.logger.Info("Docker VM deleted", "vm_id", vmID)
	return nil
}

// BootVM starts a Docker container VM
func (b *Backend) BootVM(ctx context.Context, vmID string) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	if err := b.dockerClient.ContainerStart(ctx, vm.containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_RUNNING
	b.mutex.Unlock()

	b.logger.Info("Docker VM booted", "vm_id", vmID)
	return nil
}

// ShutdownVM gracefully stops a Docker container VM
func (b *Backend) ShutdownVM(ctx context.Context, vmID string) error {
	return b.ShutdownVMWithOptions(ctx, vmID, false, 30)
}

// ShutdownVMWithOptions gracefully stops a Docker container VM with options
func (b *Backend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	timeout := int(timeoutSeconds)
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := b.dockerClient.ContainerStop(ctx, vm.containerID, stopOptions); err != nil {
		if force {
			// Force kill if graceful stop fails
			if killErr := b.dockerClient.ContainerKill(ctx, vm.containerID, "KILL"); killErr != nil {
				return fmt.Errorf("failed to force kill container: %w", killErr)
			}
		} else {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Update state
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_SHUTDOWN
	b.mutex.Unlock()

	b.logger.Info("Docker VM shutdown", "vm_id", vmID, "force", force)
	return nil
}

// PauseVM pauses a Docker container VM
func (b *Backend) PauseVM(ctx context.Context, vmID string) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	if err := b.dockerClient.ContainerPause(ctx, vm.containerID); err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	// Update state
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_PAUSED
	b.mutex.Unlock()

	b.logger.Info("Docker VM paused", "vm_id", vmID)
	return nil
}

// ResumeVM resumes a paused Docker container VM
func (b *Backend) ResumeVM(ctx context.Context, vmID string) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	if err := b.dockerClient.ContainerUnpause(ctx, vm.containerID); err != nil {
		return fmt.Errorf("failed to unpause container: %w", err)
	}

	// Update state
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_RUNNING
	b.mutex.Unlock()

	b.logger.Info("Docker VM resumed", "vm_id", vmID)
	return nil
}

// RebootVM restarts a Docker container VM
func (b *Backend) RebootVM(ctx context.Context, vmID string) error {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	timeout := int(30)
	if err := b.dockerClient.ContainerRestart(ctx, vm.containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// State remains RUNNING after reboot
	b.mutex.Lock()
	vm.state = metaldv1.VmState_VM_STATE_RUNNING
	b.mutex.Unlock()

	b.logger.Info("Docker VM rebooted", "vm_id", vmID)
	return nil
}

// GetVMInfo retrieves current VM state and configuration
func (b *Backend) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	// Get current container state
	inspect, err := b.dockerClient.ContainerInspect(ctx, vm.containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Update state based on container state
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

	// Update cached state
	b.mutex.Lock()
	vm.state = state
	b.mutex.Unlock()

	return &types.VMInfo{
		Config: vm.config,
		State:  state,
	}, nil
}

// GetVMMetrics retrieves current VM resource usage metrics
func (b *Backend) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	b.mutex.RLock()
	vm, exists := b.vms[vmID]
	b.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	// Get container stats
	stats, err := b.dockerClient.ContainerStats(ctx, vm.containerID, false)
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
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	vms := make([]types.ListableVMInfo, 0, len(b.vms))
	for _, vm := range b.vms {
		vms = append(vms, types.ListableVMInfo{
			ID:     vm.vmID,
			State:  vm.state,
			Config: vm.config,
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
