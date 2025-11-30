package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// GetGateway retrieves container status and addresses for a deployment.
//
// Finds containers by gateway ID label and returns instance information
// with host.docker.internal addresses using dynamically assigned ports.
func (d *docker) GetGateway(ctx context.Context, req backend.GetGatewayRequest) (backend.GetGatewayResponse, error) {
	d.logger.Info("getting gateway", "gateway_id", req.GatewayID)

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.gateway.id=%s", req.GatewayID)),
		),
	})
	if err != nil {
		return backend.GetGatewayResponse{}, fmt.Errorf("failed to list containers: %w", err)
	}

	res := backend.GetGatewayResponse{
		Status: backend.GATEWAY_STATUS_UNSPECIFIED,
	}

	statuses := map[container.ContainerState]int{}

	for _, c := range containers {
		count, ok := statuses[c.Status]
		if !ok {
			statuses[c.Status] = 1
		} else {
			statuses[c.Status] = count + 1

		}
	}

	// TODO this is not exhaustive not correct
	if statuses[container.StateRunning] > 0 {
		res.Status = backend.GATEWAY_STATUS_RUNNING
	} else if statuses[container.StateExited] > 0 || statuses[container.StateRemoving] > 0 {
		res.Status = backend.GATEWAY_STATUS_TERMINATING
	} else {
		res.Status = backend.GATEWAY_STATUS_PENDING
	}

	return res, nil
}
