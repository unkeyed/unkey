package environment

import (
	"errors"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// RollbackDeployment moves live traffic from one deployment back to an earlier
// one. See the proto for the contract. If from is no longer live, the caller
// decided on a state that has since changed, so the rollback is refused.
func (s *Service) RollbackDeployment(ctx restate.ObjectContext, req *hydrav1.RollbackDeploymentRequest) (*hydrav1.RollbackDeploymentResponse, error) {
	if req.GetFromDeploymentId() == req.GetToDeploymentId() {
		return nil, restate.TerminalError(errors.New("from and to must be different deployments"), 400)
	}

	deployments, err := s.loadDeployments(ctx, req.GetFromDeploymentId(), req.GetToDeploymentId())
	if err != nil {
		return nil, err
	}
	from := deployments[req.GetFromDeploymentId()]
	to := deployments[req.GetToDeploymentId()]

	if err := assert.Equal(to.AppID, from.AppID, "deployments must be in the same app"); err != nil {
		return nil, restate.TerminalError(err, 400)
	}

	if !from.CurrentDeploymentID.Valid || from.CurrentDeploymentID.String != from.ID {
		return nil, restate.TerminalError(errors.New("the deployment being rolled back from is no longer live"), 400)
	}

	if err := deploygate.CheckRollbackTarget(deploygate.RollbackInput{
		Status:              to.Status,
		DesiredState:        to.DesiredState,
		EnvironmentKind:     to.EnvironmentKind,
		CurrentDeploymentID: from.ID,
		DeploymentID:        to.ID,
	}); err != nil {
		return nil, gatefault.Terminal(err)
	}

	// The deployment traffic returns to may still carry the standby from when
	// it was demoted.
	_, err = hydrav1.NewDeploymentServiceClient(ctx, to.ID).
		ClearScheduledStateChanges().
		Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, fmt.Errorf("clear scheduled state changes: %w", err)
	}

	routeIDs, err := s.findStickyRouteIDs(ctx, to.EnvironmentID)
	if err != nil {
		return nil, err
	}

	_, err = hydrav1.NewRoutingServiceClient(ctx, to.EnvironmentID).
		SwapLiveDeployment().
		Request(&hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:      to.ID,
			FrontlineRouteIds: routeIDs,
			SetRollbackFlag:   true,
		})
	if err != nil {
		return nil, fmt.Errorf("swap live deployment: %w", err)
	}

	if err := s.insertLifecycleAudit(
		ctx,
		req.GetActor(),
		req.GetCorrelationId(),
		to,
		auditlog.DeploymentRollbackEvent,
		fmt.Sprintf("Rolled back to deployment %s", to.ID),
	); err != nil {
		return nil, fmt.Errorf("insert rollback audit log: %w", err)
	}

	logger.Info(
		"deployment rolled back",
		"environment_id", to.EnvironmentID,
		"from_deployment_id", from.ID,
		"to_deployment_id", to.ID,
		"routes", len(routeIDs),
	)
	return &hydrav1.RollbackDeploymentResponse{}, nil
}
