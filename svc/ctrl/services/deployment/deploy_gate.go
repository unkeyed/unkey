package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// ensureWorkspaceCanDeploy is the authoritative billing gate for every ctrl
// action that creates, starts, or activates compute.
func (s *Service) ensureWorkspaceCanDeploy(ctx context.Context, workspaceID string) error {
	entitlement, err := s.db.FindWorkspaceDeployEntitlement(ctx, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("workspace %q not found", workspaceID))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load workspace entitlement: %w", err))
	}

	if err := deploygate.CheckWorkspacePlan(entitlement.Plan, entitlement.PlanOverride); err != nil {
		return gatefault.Connect(err)
	}

	return gatefault.Connect(deploygate.CheckWorkspaceSpend(entitlement.SpendSuspended.Bool))
}
