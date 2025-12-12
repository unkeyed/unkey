package gatewaycontroller

import (
	"context"

	gatewayv1 "github.com/unkeyed/unkey/go/apps/krane/gateway_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *GatewayController) GetRunningGatewayIDs(ctx context.Context) <-chan string {
	gatewayIDs := make(chan string)

	go func() {

		defer close(gatewayIDs)

		gws := gatewayv1.GatewayList{} // nolint:exhaustruct
		err := c.mgr.GetClient().List(ctx, &gws, &client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					ManagedByKrane().
					ToMap(),
			),

			Namespace: "", // empty to match across all
		})

		if err != nil {
			c.logger.Error("unable to list gateways", "error", err.Error())
			return
		}

		for _, gateway := range gws.Items {

			gatewayID, ok := k8s.GetGatewayID(gateway.GetLabels())

			if !ok {
				c.logger.Warn("skipping non-gateway gateway", "name", gateway.Name)
				continue
			}
			gatewayIDs <- gatewayID
		}

	}()
	return gatewayIDs
}
