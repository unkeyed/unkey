package workspaces

import (
	"context"
	"fmt"
)

func (s *service) ChangePlan(ctx context.Context, req ChangePlanRequest) (ChangePlanResponse, error) {

	ws, found, err := s.database.FindWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return ChangePlanResponse{}, fmt.Errorf("unable to find workspace: %w", err)
	}
	if !found {
		return ChangePlanResponse{}, fmt.Errorf("workspace %s not found", req.WorkspaceId)
	}

	ws.Plan = req.Plan
	err = s.database.UpdateWorkspace(ctx, ws)
	if err != nil {
		return ChangePlanResponse{}, fmt.Errorf("unable to update workspace: %w", err)
	}
	return ChangePlanResponse{}, nil
}
