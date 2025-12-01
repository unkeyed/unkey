package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// ApplyGateway creates or updates containers for a gateway in an idempotent manner.
//
// This method ensures that the specified number of gateway containers exist.
// If containers already exist with the same gateway ID, they are reused rather than
// recreated. The operation is fully idempotent and can be safely retried.
//
// The method:
//   - Checks for existing containers with the gateway ID label
//   - Creates only missing containers (based on replica count)
//   - Skips creation if containers already exist
//   - Handles name conflicts gracefully
//
// Idempotency is guaranteed through label-based identification and name checking.
// Multiple calls with the same parameters will result in the same set of containers.
func (d *docker) ApplyGateway(ctx context.Context, req backend.ApplyGatewayRequest) error {
	d.logger.Info("creating gateway",
		"gateway_id", req.GatewayID,
		"image", req.Image,
	)

	// Ensure image exists locally (pull if not present)
	if err := d.ensureImageExists(ctx, req.Image); err != nil {
		return fmt.Errorf("failed to ensure image exists: %w", err)
	}

	// Configure port mapping
	exposedPorts := nat.PortSet{
		"8040/tcp": struct{}{},
	}

	portBindings := nat.PortMap{
		"8040/tcp": []nat.PortBinding{
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
			"unkey.gateway.id": req.GatewayID,
			"unkey.managed.by": "krane",
		},
		ExposedPorts: exposedPorts,
		Env: []string{
			fmt.Sprintf("UNKEY_WORKSPACE_ID=%s", req.WorkspaceID),
			fmt.Sprintf("UNKEY_PROJECT_ID=%s", req.ProjectID),
			fmt.Sprintf("UNKEY_ENVIRONMENT_ID=%s", req.EnvironmentID),
			fmt.Sprintf("UNKEY_GATEWAY_ID=%s", req.GatewayID),
			fmt.Sprintf("UNKEY_IMAGE=%s", req.Image),
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

	// Check existing containers for this gateway
	existingContainers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.gateway.id=%s", req.GatewayID)),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to list existing containers: %w", err)
	}

	// Map existing containers by name for quick lookup
	existingByName := make(map[string]bool)
	for _, c := range existingContainers {
		for _, name := range c.Names {
			// Docker container names have a leading slash
			existingByName[name] = true
		}
	}

	// Create containers (idempotent - skip if already exists)
	for i := range req.Replicas {
		containerName := fmt.Sprintf("%s-%d", req.GatewayID, i)

		// Check if container already exists
		if existingByName["/"+containerName] {
			d.logger.Info("container already exists, skipping", "name", containerName)
			continue
		}

		//nolint:exhaustruct // Docker SDK types have many optional fields
		resp, err := d.client.ContainerCreate(
			ctx,
			containerConfig,
			hostConfig,
			networkConfig,
			nil,
			containerName,
		)
		if err != nil {
			// Check if it's a name conflict error (in case of race condition)
			if strings.Contains(err.Error(), "Conflict") || strings.Contains(err.Error(), "already in use") {
				d.logger.Info("container already exists (race condition), skipping", "name", containerName)
				continue
			}
			return fmt.Errorf("failed to create container %s: %w", containerName, err)
		}

		//nolint:exhaustruct // Docker SDK types have many optional fields
		err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
		if err != nil {
			// Check if container is already running
			if strings.Contains(err.Error(), "already started") {
				d.logger.Info("container already running, skipping start", "name", containerName)
				continue
			}
			return fmt.Errorf("failed to start container %s: %w", containerName, err)
		}

		d.logger.Info("container created and started successfully", "name", containerName, "id", resp.ID)
	}

	return nil
}
