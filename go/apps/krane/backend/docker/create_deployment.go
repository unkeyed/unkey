package docker

import (
	"context"
	"fmt"
	"io"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/go-connections/nat"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

// CreateDeployment creates containers for a deployment with the specified replica count.
//
// Creates multiple containers with shared labels, dynamic port mapping to port 8080,
// and resource limits. Returns DEPLOYMENT_STATUS_PENDING as containers may not be
// immediately ready.
func (d *docker) CreateDeployment(ctx context.Context, req *connect.Request[kranev1.CreateDeploymentRequest]) (*connect.Response[kranev1.CreateDeploymentResponse], error) {
	deployment := req.Msg.GetDeployment()

	d.logger.Info("creating deployment",
		"deployment_id", deployment.GetDeploymentId(),
		"image", deployment.GetImage(),
	)

	// Pull the image with authentication
	d.logger.Info("pulling image", "image", deployment.GetImage())

	if d.depotToken == "" {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("depot token not configured"))
	}

	authConfig := registry.AuthConfig{
		Username: "x-token",
		// TODO: token should be generic
		Password: d.depotToken,
	}
	encodedAuth, err := registry.EncodeAuthConfig(authConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to encode auth: %w", err))
	}

	pullResp, err := d.client.ImagePull(ctx, deployment.GetImage(), image.PullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to pull image: %w", err))
	}
	defer pullResp.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, pullResp)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to complete image pull: %w", err))
	}
	d.logger.Info("image pulled successfully", "image", deployment.GetImage())

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

	for i := range req.Msg.Deployment.Replicas {
		resp, err := d.client.ContainerCreate(
			ctx,
			containerConfig,
			hostConfig,
			networkConfig,
			nil,
			fmt.Sprintf("%s-%d", deployment.GetDeploymentId(), i),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create container: %w", err))
		}

		// Start container
		err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start container: %w", err))
		}
	}

	return connect.NewResponse(&kranev1.CreateDeploymentResponse{
		Status: kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}
