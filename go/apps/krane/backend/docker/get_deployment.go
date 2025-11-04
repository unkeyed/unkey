package docker

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

// GetDeployment retrieves container status and addresses for a deployment.
//
// Finds containers by deployment ID label and returns instance information
// with host.docker.internal addresses using dynamically assigned ports.
func (d *docker) GetDeployment(ctx context.Context, req *connect.Request[kranev1.GetDeploymentRequest]) (*connect.Response[kranev1.GetDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	d.logger.Info("getting deployment", "deployment_id", deploymentID)

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.deployment.id=%s", deploymentID)),
		),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list containers: %w", err))
	}

	res := &kranev1.GetDeploymentResponse{
		Instances: []*kranev1.Instance{},
	}

	for _, c := range containers {
		d.logger.Info("container found", "container", c)

		// Determine container status
		status := kranev1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
		switch c.State {
		case container.StateRunning:
			status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING
		case container.StateExited:
			status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_TERMINATING
		case container.StateCreated:
			status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
		}

		d.logger.Info("deployment found",
			"deployment_id", deploymentID,
			"container_id", c.ID,
			"status", status.String(),
			"port", c.Ports[0].PublicPort,
		)

		res.Instances = append(res.Instances, &kranev1.Instance{
			Id:      c.ID,
			Address: fmt.Sprintf("host.docker.internal:%d", c.Ports[0].PublicPort),
			Status:  status,
		})
	}

	return connect.NewResponse(res), nil
}
