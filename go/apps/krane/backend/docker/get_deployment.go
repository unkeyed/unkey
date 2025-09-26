package docker

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/docker/docker/client"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
)

func (d *docker) GetDeployment(ctx context.Context, req *connect.Request[kranev1.GetDeploymentRequest]) (*connect.Response[kranev1.GetDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()

	d.logger.Info("getting deployment", "deployment_id", deploymentID)

	// Find container by deployment ID (using container name)
	containerJSON, err := d.client.ContainerInspect(ctx, deploymentID)
	if err != nil {
		// Check if container doesn't exist
		if client.IsErrNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", deploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to inspect container: %w", err))
	}

	// Check if this container is managed by Krane
	managedBy, exists := containerJSON.Config.Labels["unkey.managed.by"]
	if !exists || managedBy != "krane" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", deploymentID))
	}

	// Determine container status
	var status kranev1.DeploymentStatus
	if containerJSON.State.Running {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING
	} else if containerJSON.State.Dead || containerJSON.State.ExitCode != 0 {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_TERMINATING
	} else {
		status = kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	}

	// Get assigned port
	var assignedPort int32
	if bindings, exists := containerJSON.NetworkSettings.Ports["8080/tcp"]; exists && len(bindings) > 0 {
		if port, err := strconv.Atoi(bindings[0].HostPort); err == nil {
			assignedPort = int32(port)
		}
	}

	d.logger.Info("deployment found",
		"deployment_id", deploymentID,
		"container_id", containerJSON.ID,
		"status", status.String(),
		"port", assignedPort,
	)

	return connect.NewResponse(&kranev1.GetDeploymentResponse{
		Instances: []*kranev1.Instance{
			{Id: containerJSON.ID, Address: fmt.Sprintf("%s:%d", containerJSON.Name, assignedPort), Status: status},
		},
	}), nil
}
