package docker

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

// CreateGateway creates containers for a gateway with the specified replica count.
//
// Creates multiple containers with shared labels, dynamic port mapping to port 8040,
// and resource limits. Returns GATEWAY_STATUS_PENDING as containers may not be
// immediately ready.
func (d *docker) CreateGateway(ctx context.Context, req *connect.Request[kranev1.CreateGatewayRequest]) (*connect.Response[kranev1.CreateGatewayResponse], error) {
	gateway := req.Msg.GetGateway()
	d.logger.Info("creating gateway",
		"gateway_id", gateway.GetGatewayId(),
		"image", gateway.GetImage(),
	)

	// Ensure image exists locally (pull if not present)
	if err := d.ensureImageExists(ctx, gateway.GetImage()); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to ensure image exists: %w", err))
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
	cpuNanos := int64(gateway.GetCpuMillicores()) * 1_000_000      // Convert millicores to nanoseconds
	memoryBytes := int64(gateway.GetMemorySizeMib()) * 1024 * 1024 //nolint:gosec // Intentional conversion

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containerConfig := &container.Config{
		Image: gateway.GetImage(),
		Labels: map[string]string{
			"unkey.gateway.id": gateway.GetGatewayId(),
			"unkey.managed.by": "krane",
		},
		ExposedPorts: exposedPorts,
		Env: []string{
			fmt.Sprintf("UNKEY_WORKSPACE_ID=%s", gateway.GetWorkspaceId()),
			fmt.Sprintf("UNKEY_GATEWAY_ID=%s", gateway.GetGatewayId()),
			fmt.Sprintf("UNKEY_IMAGE=%s", gateway.GetImage()),
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

	for i := range req.Msg.GetGateway().GetReplicas() {
		//nolint:exhaustruct // Docker SDK types have many optional fields
		resp, err := d.client.ContainerCreate(
			ctx,
			containerConfig,
			hostConfig,
			networkConfig,
			nil,
			fmt.Sprintf("%s-%d", gateway.GetGatewayId(), i),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create container: %w", err))
		}

		//nolint:exhaustruct // Docker SDK types have many optional fields
		err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start container: %w", err))
		}
	}

	return connect.NewResponse(&kranev1.CreateGatewayResponse{
		Status: kranev1.GatewayStatus_GATEWAY_STATUS_PENDING,
	}), nil
}
