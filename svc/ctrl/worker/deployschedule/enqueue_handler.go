package deployschedule

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// Enqueue routes a deploy request to the per-branch DeployQueueService.
// This is the main entry point for all new deployments.
func (s *Service) Enqueue(ctx restate.ObjectContext, req *hydrav1.SchedulerEnqueueRequest) (*hydrav1.SchedulerEnqueueResponse, error) {
	hydrav1.NewDeployQueueServiceClient(ctx, req.GetQueueKey()).Enqueue().
		Send(&hydrav1.QueueEnqueueRequest{
			WorkspaceId:   req.GetWorkspaceId(),
			DeploymentId:  req.GetDeploymentId(),
			DeployRequest: req.GetDeployRequest(),
			IsProduction:  req.GetIsProduction(),
			Branch:        req.GetBranch(),
		})

	return &hydrav1.SchedulerEnqueueResponse{}, nil
}
