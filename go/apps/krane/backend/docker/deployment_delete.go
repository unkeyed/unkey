package docker

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// DeleteDeployment removes all containers for a deployment.
//
// Finds containers by deployment ID label and forcibly removes them with
// volumes and network links to ensure complete cleanup. This method is idempotent
// and will not fail if the containers are already deleted.
func (d *docker) DeleteDeployment(ctx context.Context, req backend.DeleteDeploymentRequest) error {
	d.logger.Info("deleting deployment", "deployment_id", req.DeploymentID)

	return d.deleteByLabels(ctx, map[string]string{
		"unkey.deployment.id": req.DeploymentID,
	})

}
