package workspaces

import (
	"context"
	"fmt"

	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

func (s *service) ChangePlan(ctx context.Context, req *workspacesv1.ChangePlanRequest) (*workspacesv1.ChangePlanResponse, error) {

	ws, found, err := s.database.FindWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return nil, fmt.Errorf("unable to find workspace: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("workspace %s not found", req.WorkspaceId)
	}

	ws.Plan = req.Plan
	err = s.database.UpdateWorkspace(ctx, ws)
	if err != nil {
		return nil, fmt.Errorf("unable to update workspace: %w", err)
	}
	return &workspacesv1.ChangePlanResponse{}, nil
}
