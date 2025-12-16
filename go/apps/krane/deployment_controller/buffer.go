package deploymentcontroller

import (
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *DeploymentController) BufferEvent(e *ctrlv1.DeploymentEvent) {
	c.events.Buffer(e)
}

func (c *DeploymentController) Changes() <-chan *ctrlv1.UpdateInstanceRequest {
	return c.changes.Consume()
}
