package deployqueue

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// OnDeployComplete is called by the deploy workflow when it finishes (success,
// failure, or cancellation). Clears the active deploy, releases the build slot
// on the workspace scheduler, and triggers ProcessNext for any remaining items.
func (s *Service) OnDeployComplete(ctx restate.ObjectContext, req *hydrav1.OnDeployCompleteRequest) (*hydrav1.OnDeployCompleteResponse, error) {
	active, _ := restate.Get[*activeDeploy](ctx, stateActive)

	// If the active deploy doesn't match (e.g. already cleared by Enqueue
	// when superseding), just skip the slot release — it was already handled.
	if active != nil && active.DeploymentID == req.GetDeploymentId() {
		restate.Set[*activeDeploy](ctx, stateActive, nil)

		// Release the build slot on the workspace scheduler.
		workspaceID, _ := restate.Get[string](ctx, stateWorkspaceID)
		hydrav1.NewDeploySchedulerServiceClient(ctx, workspaceID).ReleaseBuildSlot().
			Send(&hydrav1.ReleaseBuildSlotRequest{
				QueueKey: restate.Key(ctx),
			})

		logger.Info("deploy completed, released build slot",
			"deployment_id", req.GetDeploymentId(),
			"queue_key", restate.Key(ctx),
		)
	}

	// Process next item in the queue, if any.
	hydrav1.NewDeployQueueServiceClient(ctx, restate.Key(ctx)).ProcessNext().
		Send(&hydrav1.ProcessNextRequest{})

	return &hydrav1.OnDeployCompleteResponse{}, nil
}
