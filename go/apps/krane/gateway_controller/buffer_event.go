package gatewaycontroller

import (
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *GatewayController) BufferEvent(evt *ctrlv1.GatewayEvent) {
	c.events.Buffer(evt)
}
