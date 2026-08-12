package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// ensureWorkspaceCanDeploy is the authoritative billing gate for every ctrl
// action that creates, starts, or activates compute. Plan enforcement supports
// observe mode during rollout; spend-cap suspension always blocks the action.
func (s *Service) ensureWorkspaceCanDeploy(ctx context.Context, workspaceID, action string) error {
	entitlement, err := s.db.FindWorkspaceDeployEntitlement(ctx, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("workspace %q not found", workspaceID))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load workspace entitlement: %w", err))
	}

	if planErr := deploygate.CheckWorkspacePlan(entitlement.Plan, entitlement.PlanOverride); planErr != nil {
		if s.enforceDeployGate {
			return gatefault.Connect(planErr)
		}
		logger.Warn("deploy gate would block deployment action",
			"event", "deploy_gate.would_block",
			"workspaceId", workspaceID,
			"action", action,
		)
	}

	return gatefault.Connect(deploygate.CheckWorkspaceSpend(entitlement.SpendSuspended.Bool))
}
