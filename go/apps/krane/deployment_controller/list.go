package deploymentcontroller

import (
	"context"

	deploymentv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetScheduledDeploymentIDs returns a channel of all deployment IDs managed by krane.
//
// This method lists all Deployment CRDs across all namespaces and
// streams their deployment IDs through the returned channel. The channel is
// closed when all deployments have been processed.
//
// Parameters:
//   - ctx: Context for the listing operation
//
// Returns a channel that emits deployment IDs. Errors are logged but
// do not affect channel operation.
func (c *DeploymentController) GetScheduledDeploymentIDs(ctx context.Context) <-chan string {
	deploymentIDs := make(chan string)

	go func() {

		defer close(deploymentIDs)

		cursor := ""
		for {

			deployments := deploymentv1.DeploymentList{} // nolint:exhaustruct
			err := c.client.List(ctx, &deployments, &client.ListOptions{
				LabelSelector: k8s.NewLabels().
					ManagedByKrane().
					ToSelector(),
				Continue:  cursor,
				Namespace: "", // empty to match across all
			})

			if err != nil {
				c.logger.Error("unable to list deployments", "error", err.Error())
				return
			}

			for _, deployment := range deployments.Items {

				deploymentID, ok := k8s.GetDeploymentID(deployment.GetLabels())

				if !ok {
					c.logger.Warn("skipping non-deployment deployment", "name", deployment.Name)
					continue
				}
				deploymentIDs <- deploymentID
			}
			cursor = deployments.Continue
			if cursor == "" {
				break
			}
		}

	}()
	return deploymentIDs
}
