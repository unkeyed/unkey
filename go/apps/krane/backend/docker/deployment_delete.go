package docker

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

// DeleteDeployment removes all containers for a deployment.
//
// Finds containers by deployment ID label and forcibly removes them with
// volumes and network links to ensure complete cleanup.
func (d *docker) DeleteDeployment(ctx context.Context, req *connect.Request[kranev1.DeleteDeploymentRequest]) (*connect.Response[kranev1.DeleteDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()

	d.logger.Info("getting deployment", "deployment_id", deploymentID)

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Size:   false,
		Latest: false,
		Since:  "",
		Before: "",
		Limit:  0,
		All:    true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.deployment.id=%s", deploymentID)),
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
	return connect.NewResponse(&kranev1.DeleteDeploymentResponse{}), nil
}
