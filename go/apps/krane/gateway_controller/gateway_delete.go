package gatewaycontroller

import (
	"context"
	"fmt"

	gatewayv1 "github.com/unkeyed/unkey/go/apps/krane/gateway_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *GatewayController) DeleteGateway(ctx context.Context, req *ctrlv1.DeleteGateway) error {

	c.logger.Info("deleting gateway",
		"gateway_id", req.GetGatewayId(),
	)

	gatewayList := gatewayv1.GatewayList{} //nolint:exhaustruct
	if err := c.mgr.GetClient().List(ctx, &gatewayList,
		&client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(
				k8s.NewLabels().
					ManagedByKrane().
					ToMap(),
			),

			Namespace: "", // empty to match across all
		},
	); err != nil {
		return fmt.Errorf("failed to list gateways: %w", err)
	}

	if len(gatewayList.Items) == 0 {

		c.logger.Debug("gateway had no CRD configured", "gateway_id", req.GetGatewayId())
		return nil
	}

	for _, gateway := range gatewayList.Items {
		err := c.mgr.GetClient().Delete(ctx, &gateway)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete gateway resource %s: %w", gateway.Name, err)
		}
	}

	c.logger.Info("gateway deleted successfully",
		"gateway_id", req.GetGatewayId(),
	)

	return nil
}
