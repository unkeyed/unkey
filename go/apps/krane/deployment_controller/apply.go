package deploymentcontroller

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *DeploymentController) apply() {

	for e := range c.events.Consume() {
		ctx := context.Background()
		switch x := e.GetEvent().(type) {
		case *ctrlv1.DeploymentEvent_Apply:
			if err := c.ApplyDeployment(ctx, x.Apply); err != nil {
				c.logger.Error("unable to apply deployment", "deployment_id", x.Apply.GetDeploymentId())

			}
		case *ctrlv1.DeploymentEvent_Delete:
			if err := c.DeleteDeployment(ctx, x.Delete); err != nil {
				c.logger.Error("unable to delete deployment", "deployment_id", x.Delete.GetDeploymentId())
			}
		}

	}
}
