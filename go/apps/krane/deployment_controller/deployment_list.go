package deploymentcontroller

import (
	"context"

	deploymentv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DeploymentController) GetScheduledDeploymentIDs(ctx context.Context) <-chan string {
	deploymentIDs := make(chan string)

	go func() {

		defer close(deploymentIDs)

		gws := deploymentv1.UnkeyDeploymentList{} // nolint:exhaustruct
		err := c.manager.GetClient().List(ctx, &gws, &client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					ManagedByKrane().
					ToMap(),
			),

			Namespace: "", // empty to match across all
		})

		if err != nil {
			c.logger.Error("unable to list deployments", "error", err.Error())
			return
		}

		for _, deployment := range gws.Items {

			deploymentID, ok := k8s.GetDeploymentID(deployment.GetLabels())

			if !ok {
				c.logger.Warn("skipping non-deployment deployment", "name", deployment.Name)
				continue
			}
			deploymentIDs <- deploymentID
		}

	}()
	return deploymentIDs
}
