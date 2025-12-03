package gatewaycontroller

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *GatewayController) apply() {

	for e := range c.events.Consume() {
		ctx := context.Background()
		switch x := e.GetEvent().(type) {
		case *ctrlv1.GatewayEvent_Apply:
			if err := c.ApplyGateway(ctx, x.Apply); err != nil {
				c.logger.Error("unable to apply gateway", "gateway_id", x.Apply.GetGatewayId())
			}
		case *ctrlv1.GatewayEvent_Delete:
			if err := c.DeleteGateway(ctx, x.Delete); err != nil {
				c.logger.Error("unable to delete gateway", "gateway_id", x.Delete.GetGatewayId())
			}
		}

	}
}
