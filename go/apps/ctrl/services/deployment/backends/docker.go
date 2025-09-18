package backends

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// DockerBackend implements DeploymentBackend using Docker containers
type DockerBackend struct {
	logger          logging.Logger
	dockerClient    *client.Client
	deployments     map[string]*dockerDeployment
	mutex           sync.RWMutex
	isRunningDocker bool // Whether this service is running in Docker
}

type dockerDeployment struct {
	DeploymentID string
	ContainerIDs []string
	VMIDs        []string
	Image        string
	CreatedAt    time.Time
}

// NewDockerBackend creates a new Docker backend
func NewDockerBackend(logger logging.Logger, isRunningDocker bool) (*DockerBackend, error) {
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

	return &DockerBackend{
		logger:          logger.With("backend", "docker"),
		dockerClient:    dockerClient,
		deployments:     make(map[string]*dockerDeployment),
		isRunningDocker: isRunningDocker,
	}, nil
}

// CreateDeployment creates Docker containers for the deployment
func (d *DockerBackend) CreateDeployment(ctx context.Context, deploymentID string, image string, vmCount uint32) ([]string, error) {
	d.logger.Info("creating Docker deployment",
		"deployment_id", deploymentID,
		"image", image,
		"vm_count", vmCount)

	// Pull image if not present
	if err := d.pullImageIfNeeded(ctx, image); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	deployment := &dockerDeployment{
		DeploymentID: deploymentID,
		Image:        image,
		CreatedAt:    time.Now(),
		ContainerIDs: make([]string, 0, vmCount),
		VMIDs:        make([]string, 0, vmCount),
	}

	// Create containers
	for i := range vmCount {
		vmID := uid.New("vm")
		containerName := fmt.Sprintf("unkey-%s-%s", deploymentID, vmID)

		containerID, err := d.createContainer(ctx, containerName, image, vmID, deploymentID)
		if err != nil {
			// Clean up any created containers on failure
			d.cleanupDeployment(ctx, deployment)
			return nil, fmt.Errorf("failed to create container %d: %w", i, err)
		}

		deployment.ContainerIDs = append(deployment.ContainerIDs, containerID)
		deployment.VMIDs = append(deployment.VMIDs, vmID)

		// Start the container
		if err := d.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			d.cleanupDeployment(ctx, deployment)
			return nil, fmt.Errorf("failed to start container %s: %w", containerID, err)
		}

		d.logger.Info("container started",
			"vm_id", vmID,
			"container_id", containerID,
			"deployment_id", deploymentID)
	}

	// Store deployment
	d.mutex.Lock()
	d.deployments[deploymentID] = deployment
	d.mutex.Unlock()

	return deployment.VMIDs, nil
}

// GetDeploymentStatus returns the status of deployment VMs
func (d *DockerBackend) GetDeploymentStatus(ctx context.Context, deploymentID string) ([]*metaldv1.GetDeploymentResponse_Vm, error) {
	d.mutex.RLock()
	deployment, exists := d.deployments[deploymentID]
	d.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}

	vms := make([]*metaldv1.GetDeploymentResponse_Vm, 0, len(deployment.ContainerIDs))

	for i, containerID := range deployment.ContainerIDs {
		inspect, err := d.dockerClient.ContainerInspect(ctx, containerID)
		if err != nil {
			d.logger.Error("failed to inspect container",
				"container_id", containerID,
				"error", err)
			continue
		}

		// Determine VM state from container state
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

		// Get host and port from container
		var host string
		port := int32(8080) // Container internal port

		// Always use container IP when available (works from inside and outside Docker)
		if inspect.NetworkSettings != nil {
			// Try legacy IPAddress field first
			if inspect.NetworkSettings.IPAddress != "" {
				host = inspect.NetworkSettings.IPAddress
				port = int32(8080)
			} else if inspect.NetworkSettings.Networks != nil {
				// Try to get IP from any network
				for _, network := range inspect.NetworkSettings.Networks {
					if network.IPAddress != "" {
						host = network.IPAddress
						port = int32(8080)
						break
					}
				}
			}
		}

		// If no container IP found, fall back to external access
		if host == "" {
			// Fallback: use external access method
			if d.isRunningDocker {
				host = "host.docker.internal"
			} else {
				host = "localhost"
			}
			// Get the mapped port for external access
			if inspect.NetworkSettings != nil && inspect.NetworkSettings.Ports != nil {
				for containerPort, bindings := range inspect.NetworkSettings.Ports {
					if strings.Contains(string(containerPort), "8080") && len(bindings) > 0 {
						if p, err := strconv.Atoi(bindings[0].HostPort); err == nil {
							port = int32(p)
						}
					}
				}
			}
		}

		vms = append(vms, &metaldv1.GetDeploymentResponse_Vm{
			Id:    deployment.VMIDs[i],
			State: state,
			Host:  host,
			Port:  uint32(port),
		})
	}

	return vms, nil
}

// DeleteDeployment removes all containers in a deployment
func (d *DockerBackend) DeleteDeployment(ctx context.Context, deploymentID string) error {
	d.mutex.Lock()
	deployment, exists := d.deployments[deploymentID]
	if exists {
		delete(d.deployments, deploymentID)
	}
	d.mutex.Unlock()

	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	return d.cleanupDeployment(ctx, deployment)
}

// Type returns the backend type
func (d *DockerBackend) Type() string {
	return BackendTypeDocker
}

// Helper methods

func (d *DockerBackend) pullImageIfNeeded(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, _, inspectErr := d.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if inspectErr == nil {
		d.logger.Info("image found locally", "image", imageName)
		return nil
	}

	// Only attempt to pull if the error indicates the image is not found
	// For other errors (e.g., permission issues, Docker daemon problems), return immediately
	if !errdefs.IsNotFound(inspectErr) {
		return fmt.Errorf("failed to inspect image %s: %w", imageName, inspectErr)
	}

	// Image not found locally, attempt to pull it
	d.logger.Info("pulling image", "image", imageName)
	reader, pullErr := d.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if pullErr != nil {
		// If pull fails, return the original inspect error for better context
		return fmt.Errorf("image %s not found locally and pull failed: inspect error: %v, pull error: %w", imageName, inspectErr, pullErr)
	}
	defer reader.Close()

	// Read the output to ensure pull completes
	_, err := io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	d.logger.Info("image pulled successfully", "image", imageName)
	return nil
}

func (d *DockerBackend) createContainer(ctx context.Context, name string, imageName string, vmID string, deploymentID string) (string, error) {
	config := &container.Config{
		Image: imageName,
		Labels: map[string]string{
			"unkey.vm.id":                         vmID,
			"unkey.deployment.id":                 deploymentID,
			"unkey.managed.by":                    "ctrl-fallback",
			"com.docker.compose.project":          "unkey_deployments",
			"com.docker.compose.service":          fmt.Sprintf("vm_%s", vmID),
			"com.docker.compose.container-number": "1",
		},
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
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
		Resources: container.Resources{
			Memory:   1024 * 1024 * 1024, // 1GB
			NanoCPUs: 1000000000,         // 1 CPU
		},
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"unkey_default": {},
		},
	}

	resp, err := d.dockerClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

func (d *DockerBackend) cleanupDeployment(ctx context.Context, deployment *dockerDeployment) error {
	var lastErr error
	for _, containerID := range deployment.ContainerIDs {
		// Stop container
		if err := d.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			d.logger.Error("failed to stop container",
				"container_id", containerID,
				"error", err)
			lastErr = err
		}

		// Remove container
		if err := d.dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{
			Force: true,
		}); err != nil {
			d.logger.Error("failed to remove container",
				"container_id", containerID,
				"error", err)
			lastErr = err
		}
	}
	return lastErr
}
