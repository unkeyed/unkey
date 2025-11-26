package docker

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

// DeleteGateway removes all containers for a gateway.
//
// Finds containers by gateway ID label and forcibly removes them with
// volumes and network links to ensure complete cleanup.
func (d *docker) DeleteGateway(ctx context.Context, req *connect.Request[kranev1.DeleteGatewayRequest]) (*connect.Response[kranev1.DeleteGatewayResponse], error) {
	gatewayID := req.Msg.GetGatewayId()

	d.logger.Info("getting gateway", "gateway_id", gatewayID)

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Size:   false,
		Latest: false,
		Since:  "",
		Before: "",
		Limit:  0,
		All:    true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.gateway.id=%s", gatewayID)),
		),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list containers: %w", err))
	}

	for _, c := range containers {
		err := d.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   true,
			Force:         true,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove container: %w", err))
		}
	}
	return connect.NewResponse(&kranev1.DeleteGatewayResponse{}), nil
}
