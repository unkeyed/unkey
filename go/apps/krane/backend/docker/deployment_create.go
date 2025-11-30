package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// CreateDeployment creates containers for a deployment with the specified replica count.
//
// Creates multiple containers with shared labels, dynamic port mapping to port 8080,
// and resource limits. Returns DEPLOYMENT_STATUS_PENDING as containers may not be
// immediately ready.
func (d *docker) CreateDeployment(ctx context.Context, req backend.CreateDeploymentRequest) error {

	d.logger.Info("creating deployment",
		"deployment_id", req.DeploymentID,
		"image", req.Image,
	)

	// Ensure image exists locally (pull if not present)
	if err := d.ensureImageExists(ctx, req.Image); err != nil {
		return fmt.Errorf("failed to ensure image exists: %w", err)
	}

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
	cpuNanos := int64(req.CpuMillicores) * 1_000_000      // Convert millicores to nanoseconds
	memoryBytes := int64(req.MemorySizeMib) * 1024 * 1024 //nolint:gosec // Intentional conversion

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containerConfig := &container.Config{
		Image: req.Image,
		Labels: map[string]string{
			"unkey.deployment.id": req.DeploymentID,
			"unkey.managed.by":    "krane",
		},
		ExposedPorts: exposedPorts,
		Env: []string{
			fmt.Sprintf("UNKEY_PROJECT_ID=%s", req.ProjectID),
			fmt.Sprintf("UNKEY_ENVIRONMENT_ID=%s", req.EnvironmentID),
			fmt.Sprintf("UNKEY_DEPLOYMENT_ID=%s", req.DeploymentID),
		},
	}

	//nolint:exhaustruct // Docker SDK types have many optional fields
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

	//nolint:exhaustruct // Docker SDK types have many optional fields
	networkConfig := &network.NetworkingConfig{}

	// Create container

	for i := range req.Replicas {
		//nolint:exhaustruct // Docker SDK types have many optional fields
		resp, err := d.client.ContainerCreate(
			ctx,
			containerConfig,
			hostConfig,
			networkConfig,
			nil,
			fmt.Sprintf("%s-%d", req.DeploymentID, i),
		)
		if err != nil {
			return fmt.Errorf("failed to create container: %w", err)
		}

		//nolint:exhaustruct // Docker SDK types have many optional fields
		err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
		if err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
	}

	return nil
}
