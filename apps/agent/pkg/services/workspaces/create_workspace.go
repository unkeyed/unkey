package workspaces

import (
	"context"

	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func (s *service) CreateWorkspace(ctx context.Context, req *workspacesv1.CreateWorkspaceRequest) (*workspacesv1.CreateWorkspaceResponse, error) {

	ws := &workspacesv1.Workspace{
		WorkspaceId: uid.Workspace(),
		Name:        req.Name,
		TenantId:    req.TenantId,
		Plan:        req.Plan,
	}

	err := s.database.InsertWorkspace(ctx, ws)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	return &workspacesv1.CreateWorkspaceResponse{WorkspaceId: ws.WorkspaceId}, nil
}
