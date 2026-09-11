package environment

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// PromoteDeployment makes one deployment the environment's live deployment.
// See the proto for the contract. The gate check and the swap share the key,
// so nothing moves the live pointer between them.
func (s *Service) PromoteDeployment(ctx restate.ObjectContext, req *hydrav1.PromoteDeploymentRequest) (*hydrav1.PromoteDeploymentResponse, error) {
	deployments, err := s.loadDeployments(ctx, req.GetDeploymentId())
	if err != nil {
		return nil, err
	}
	deployment := deployments[req.GetDeploymentId()]

	if err := deploygate.CheckPromoteTarget(deploygate.PromoteInput{
		Status:              deployment.Status,
		DesiredState:        deployment.DesiredState,
		EnvironmentKind:     deployment.EnvironmentKind,
		CurrentDeploymentID: deployment.CurrentDeploymentID.String,
		DeploymentID:        deployment.ID,
		IsRolledBack:        deployment.IsRolledBack,
	}); err != nil {
		return nil, gatefault.Terminal(err)
	}

	confirmingRollback := deployment.IsRolledBack && deployment.CurrentDeploymentID.String == deployment.ID

	var routeIDs []string
	if !confirmingRollback {
		routeIDs, err = s.findStickyRouteIDs(ctx, deployment.EnvironmentID)
		if err != nil {
			return nil, err
		}
	}

	// With no routes the swap only clears is_rolled_back.
	swap, err := hydrav1.NewRoutingServiceClient(ctx, deployment.EnvironmentID).
		SwapLiveDeployment().
		Request(&hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:      deployment.ID,
			FrontlineRouteIds: routeIDs,
			SetRollbackFlag:   false,
		})
	if err != nil {
		return nil, fmt.Errorf("swap live deployment: %w", err)
	}

	// The deployment may still carry the standby from when it was demoted.
	_, err = hydrav1.NewDeploymentServiceClient(ctx, deployment.ID).
		ClearScheduledStateChanges().
		Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, fmt.Errorf("clear scheduled state changes: %w", err)
	}

	demotedID, err := s.demotedDeploymentID(ctx, deployment, swap.GetPreviousDeploymentId(), confirmingRollback)
	if err != nil {
		return nil, err
	}
	if demotedID != "" {
		hydrav1.NewDeploymentServiceClient(ctx, demotedID).
			ScheduleDesiredStateChange().
			Send(&hydrav1.ScheduleDesiredStateChangeRequest{
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
				DelayMillis: standbyDelay.Milliseconds(),
			})
	}

	if err := s.insertLifecycleAudit(
		ctx,
		req.GetActor(),
		req.GetCorrelationId(),
		deployment,
		auditlog.DeploymentPromoteEvent,
		fmt.Sprintf("Promoted deployment %s", deployment.ID),
	); err != nil {
		return nil, fmt.Errorf("insert promote audit log: %w", err)
	}

	logger.Info("deployment promoted",
		"environment_id", deployment.EnvironmentID,
		"deployment_id", deployment.ID,
		"demoted_deployment_id", demotedID,
		"confirm_rollback", confirmingRollback,
	)
	return &hydrav1.PromoteDeploymentResponse{}, nil
}

// demotedDeploymentID is the deployment that lost live traffic. Confirming a
// rollback swaps nothing, so it is the newest ready deployment other than the
// promoted one.
func (s *Service) demotedDeploymentID(
	ctx restate.ObjectContext,
	promoted db.FindDeploymentWithEnvironmentAndAppRow,
	previous string,
	confirmingRollback bool,
) (string, error) {
	if !confirmingRollback {
		return previous, nil
	}

	demoted, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return s.db.FindLatestReadyDeploymentByAppAndEnv(runCtx, db.FindLatestReadyDeploymentByAppAndEnvParams{
			AppID:         promoted.AppID,
			EnvironmentID: promoted.EnvironmentID,
			ExcludeID:     promoted.ID,
		})
	}, restate.WithName("find demoted deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return "", fmt.Errorf("find demoted deployment: %w", err)
	}
	return demoted, nil
}
