package docker

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

func (d *docker) CreateDeployment(ctx context.Context, req *connect.Request[kranev1.CreateDeploymentRequest]) (*connect.Response[kranev1.CreateDeploymentResponse], error) {
	deployment := req.Msg.GetDeployment()

	d.logger.Info("creating deployment",
		"deployment_id", deployment.GetDeploymentId(),
		"image", deployment.GetImage(),
	)

	// Configure port mapping
	exposedPorts := nat.PortSet{
		"8080/tcp": struct{}{},
	}

	portBindings := nat.PortMap{
		"8080/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: "0", // Docker will assign a random available port
			},
		},
	}

	// Configure resource limits
	cpuNanos := int64(deployment.GetCpuMillicores()) * 1_000_000 // Convert millicores to nanoseconds
	memoryBytes := int64(deployment.GetMemorySizeMib()) * 1024 * 1024

	// Container configuration
	containerConfig := &container.Config{
		Image: deployment.GetImage(),
		Labels: map[string]string{
			"unkey.deployment.id": deployment.GetDeploymentId(),
			"unkey.managed.by":    "krane",
		},
		ExposedPorts: exposedPorts,
		Env: []string{
			fmt.Sprintf("DEPLOYMENT_ID=%s", deployment.GetDeploymentId()),
		},
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			NanoCPUs: cpuNanos,
			Memory:   memoryBytes,
		},
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{}

	// Create container
	containerName := deployment.GetDeploymentId()
	resp, err := d.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create container: %w", err))
	}

	// Start container
	err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		// Clean up container if start fails
		_ = d.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start container: %w", err))
	}

	// Get container info to retrieve the assigned port
	containerJSON, err := d.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to inspect container: %w", err))
	}

	var assignedPort int32
	if bindings, exists := containerJSON.NetworkSettings.Ports["8080/tcp"]; exists && len(bindings) > 0 {
		if port, err := strconv.Atoi(bindings[0].HostPort); err == nil {
			assignedPort = int32(port)
		}
	}

	d.logger.Info("container created and started",
		"container_id", resp.ID,
		"deployment_id", deployment.GetDeploymentId(),
		"assigned_port", assignedPort,
	)

	return connect.NewResponse(&kranev1.CreateDeploymentResponse{
		Status: kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}
