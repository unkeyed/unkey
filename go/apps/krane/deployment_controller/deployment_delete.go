package deploymentcontroller

import (
	"context"
	"fmt"

	deploymentv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteDeployment removes Deployment CRDs with the specified deployment ID.
//
// This method finds and deletes all Deployment custom resources matching the
// given deployment ID across all namespaces. The controller-runtime reconciler
// will handle the actual Kubernetes resource cleanup based on these CRD deletions.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: Deployment deletion request containing the deployment ID
//
// Returns an error if the listing operation fails or if any
// Deployment CRD cannot be deleted.
func (c *DeploymentController) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error {

	c.logger.Info("deleting deployment",
		"deployment_id", req.GetDeploymentId(),
	)

	deploymentList := deploymentv1.DeploymentList{} //nolint:exhaustruct
	if err := c.client.List(ctx, &deploymentList,
		&client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					ManagedByKrane().
					ToMap(),
			),

			Namespace: "", // empty to match across all
		},
	); err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deploymentList.Items) == 0 {

		c.logger.Debug("deployment had no CRD configured", "deployment_id", req.GetDeploymentId())
		return nil
	}

	for _, deployment := range deploymentList.Items {
		err := c.client.Delete(ctx, &deployment)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete deployment resource %s: %w", deployment.Name, err)
		}
	}

	c.logger.Info("deployment deleted successfully",
		"deployment_id", req.GetDeploymentId(),
	)

	return nil
}
