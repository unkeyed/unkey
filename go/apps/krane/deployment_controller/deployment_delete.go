package deploymentcontroller

import (
	"context"
	"fmt"

	v1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteDeployment removes UnkeyDeployment CRDs with the specified deployment ID.
// The controller-runtime reconciler will handle the actual Kubernetes resource cleanup.
func (c *DeploymentController) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error {
	deploymentID := req.GetDeploymentId()

	c.logger.Info("deleting UnkeyDeployment",
		"deployment_id", deploymentID,
	)

	// List all UnkeyDeployment CRDs with matching deployment ID across all namespaces
	deploymentList := v1.UnkeyDeploymentList{} //nolint:exhaustruct
	if err := c.manager.GetClient().List(ctx, &deploymentList,
		&client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					DeploymentID(deploymentID).
					ManagedByKrane().
					ToMap(),
			),
			Namespace: "", // empty to match across all namespaces
		},
	); err != nil {
		return fmt.Errorf("failed to list UnkeyDeployments: %w", err)
	}

	if len(deploymentList.Items) == 0 {
		c.logger.Debug("no UnkeyDeployment CRDs found for deployment", "deployment_id", deploymentID)
		return nil
	}

	// Delete all found UnkeyDeployment CRDs
	for _, deployment := range deploymentList.Items {
		err := c.manager.GetClient().Delete(ctx, &deployment)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete UnkeyDeployment %s: %w", deployment.Name, err)
		}
		c.logger.Debug("deleted UnkeyDeployment CRD",
			"name", deployment.Name,
			"namespace", deployment.Namespace,
			"deployment_id", deploymentID,
		)
	}

	c.logger.Info("UnkeyDeployment(s) deleted successfully",
		"deployment_id", deploymentID,
		"count", len(deploymentList.Items),
	)

	return nil
}
