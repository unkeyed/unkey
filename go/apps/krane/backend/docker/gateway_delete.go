package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// DeleteGateway removes all containers for a gateway.
//
// Finds containers by gateway ID label and forcibly removes them with
// volumes and network links to ensure complete cleanup.
func (d *docker) DeleteGateway(ctx context.Context, req backend.DeleteGatewayRequest) error {

	d.logger.Info("getting gateway", "gateway_id", req.GatewayID)

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Size:   false,
		Latest: false,
		Since:  "",
		Before: "",
		Limit:  0,
		All:    true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.gateway.id=%s", req.GatewayID)),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		err := d.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   true,
			Force:         true,
		})
		if err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}
	return nil
}
