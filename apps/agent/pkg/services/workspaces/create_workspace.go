package workspaces

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func (s *service) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (CreateWorkspaceResponse, error) {

	ws := entities.Workspace{
		Id:       uid.Workspace(),
		Name:     req.Name,
		TenantId: req.TenantId,
		Plan:     req.Plan,
	}

	err := s.database.InsertWorkspace(ctx, ws)
	if err != nil {
		return CreateWorkspaceResponse{}, fmt.Errorf("unable to create workspace: %w", err)
	}
	return CreateWorkspaceResponse{ws}, nil
}
