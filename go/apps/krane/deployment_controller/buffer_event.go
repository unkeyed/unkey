package deploymentcontroller

import (
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *DeploymentController) BufferEvent(evt *ctrlv1.DeploymentEvent) {
	c.events.Buffer(evt)
}
