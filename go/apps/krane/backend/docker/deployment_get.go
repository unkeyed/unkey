package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// GetDeployment retrieves container status and addresses for a deployment.
//
// Finds containers by deployment ID label and returns instance information
// with host.docker.internal addresses using dynamically assigned ports.
func (d *docker) GetDeployment(ctx context.Context, req backend.GetDeploymentRequest) (backend.GetDeploymentResponse, error) {
	d.logger.Info("getting deployment", "deployment_id", req.DeploymentID)

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("unkey.deployment.id=%s", req.DeploymentID)),
		),
	})
	if err != nil {
		return backend.GetDeploymentResponse{}, fmt.Errorf("failed to list containers: %w", err)
	}

	res := backend.GetDeploymentResponse{
		Instances: []backend.Instance{},
	}

	for _, c := range containers {
		d.logger.Info("container found", "container", c)

		// Determine container status
		status := backend.DEPLOYMENT_STATUS_UNSPECIFIED
		switch c.State {
		case container.StateRunning:
			status = backend.DEPLOYMENT_STATUS_RUNNING
		case container.StateExited:
			status = backend.DEPLOYMENT_STATUS_TERMINATING
		case container.StateCreated:
			status = backend.DEPLOYMENT_STATUS_RUNNING
		}

		d.logger.Info("deployment found",
			"deployment_id", req.DeploymentID,
			"container_id", c.ID,
			"status", status,
			"port", c.Ports[0].PublicPort,
		)

		res.Instances = append(res.Instances, backend.Instance{
			Id:      c.ID,
			Address: fmt.Sprintf("host.docker.internal:%d", c.Ports[0].PublicPort),
			Status:  status,
		})
	}

	return res, nil
}
