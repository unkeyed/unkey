package gatewaycontroller

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *GatewayController) GetRunningGatewayIds(ctx context.Context) <-chan string {
	gatewayIDs := make(chan string)

	go func() {
		defer close(gatewayIDs)

		cursor := ""
		for {

			deployments, err := c.clientset.AppsV1().Deployments(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.FormatLabels(k8s.NewLabels().
					ManagedByKrane().
					ComponentGateway().
					ToMap()),
				Continue: cursor,
				Limit:    100,
			})
			if err != nil {
				c.logger.Error("unable to list deployments", "error", err.Error())
				return
			}

			cursor = deployments.GetContinue()

			for _, sfs := range deployments.Items {

				gatewayID, ok := k8s.GetGatewayID(sfs.GetLabels())

				if !ok {
					c.logger.Warn("skipping non-Deployment sfs", "name", sfs.Name)
					continue
				}
				gatewayIDs <- gatewayID
			}
			if cursor == "" {
				return
			}

		}
	}()

	return gatewayIDs
}
