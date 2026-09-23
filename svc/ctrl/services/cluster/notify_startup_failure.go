package cluster

import (
	"context"
	"fmt"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) notifyStartupFailure(ctx context.Context, event *ctrlv1.InstanceEvent) error {
	var state hydrav1.NotifyReadinessRequest_State
	switch {
	case event.GetTerminated().GetCause() == ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED:
		state = hydrav1.NotifyReadinessRequest_STATE_OUT_OF_MEMORY
	case event.GetTerminated().GetCause() == ctrlv1.TerminationCause_TERMINATION_CAUSE_CONTAINER_CANNOT_RUN:
		state = hydrav1.NotifyReadinessRequest_STATE_CONTAINER_CANNOT_RUN
	case event.GetWaiting().GetCause() == ctrlv1.WaitingCause_WAITING_CAUSE_CRASH_LOOP_BACK_OFF:
		state = hydrav1.NotifyReadinessRequest_STATE_CRASH_LOOP_BACK_OFF
	case event.GetWaiting().GetCause() == ctrlv1.WaitingCause_WAITING_CAUSE_INVALID_IMAGE_NAME:
		state = hydrav1.NotifyReadinessRequest_STATE_INVALID_IMAGE_NAME
	default:
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	deploymentID := event.GetDeploymentId()
	deployment, err := s.db.FindDeploymentById(ctx, deploymentID)
	if db.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load deployment for startup failure notification: %w", err)
	}
	if deployment.Status != dbtype.DeploymentsStatusDeploying {
		return nil
	}
	if !deployment.InvocationID.Valid {
		return fmt.Errorf("deployment %s invocation ID is not yet available", deploymentID)
	}

	// Wake also uses deploying after the original Deploy workflow completed.
	live, err := s.restateAdmin.FindLiveInvocations(ctx, []string{deployment.InvocationID.String})
	if err != nil {
		return fmt.Errorf("check deployment liveness for startup failure notification: %w", err)
	}
	if !live[deployment.InvocationID.String] {
		return nil
	}
	_, err = hydrav1.NewDeployWorkflowIngressClient(s.restate, deploymentID).NotifyReadiness().Send(ctx, &hydrav1.NotifyReadinessRequest{
		State: state,
	})
	if err != nil {
		return fmt.Errorf("notify deployment of startup failure: %w", err)
	}
	logger.Info("notified deployment of startup failure", "deployment_id", deploymentID, "state", state)
	return nil
}
