package docker

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/array"
)

// GetGateway retrieves container status and addresses for a deployment.
//
// Finds containers by gateway ID label and returns instance information
// with host.docker.internal addresses using dynamically assigned ports.
func (d *docker) GetGateway(ctx context.Context, req *connect.Request[kranev1.GetGatewayRequest]) (*connect.Response[kranev1.GetGatewayResponse], error) {
	gatewayID := req.Msg.GetGatewayId()
	d.logger.Info("getting gateway", "gateway_id", gatewayID)

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.gateway.id=%s", gatewayID)),
		),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list containers: %w", err))
	}

	res := &kranev1.GetGatewayResponse{
		// This is pretty nasty, we just return a random container's address, but it's better than nothing
		// Docker is really just meant for development and testing purposes, not production use.
		Address:   fmt.Sprintf("host.docker.internal:%d", array.Random(containers).Ports[0].PublicPort),
		Instances: []*kranev1.GatewayInstance{},
	}

	for _, c := range containers {
		d.logger.Info("container found", "container", c)

		// Determine container status
		status := kranev1.GatewayStatus_GATEWAY_STATUS_UNSPECIFIED
		switch c.State {
		case container.StateRunning:
			status = kranev1.GatewayStatus_GATEWAY_STATUS_RUNNING
		case container.StateExited:
			status = kranev1.GatewayStatus_GATEWAY_STATUS_TERMINATING
		case container.StateCreated:
			status = kranev1.GatewayStatus_GATEWAY_STATUS_PENDING
		}

		d.logger.Info("gateway found",
			"gateway_id", gatewayID,
			"container_id", c.ID,
			"status", status.String(),
			"port", c.Ports[0].PublicPort,
		)

		res.Instances = append(res.Instances, &kranev1.GatewayInstance{
			Id:      c.ID,
			Address: fmt.Sprintf("host.docker.internal:%d", c.Ports[0].PublicPort),
			Status:  status,
		})
	}

	return connect.NewResponse(res), nil
}
