package workspaces

import (
	"context"
	"fmt"

	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

func (s *service) FindWorkspace(ctx context.Context, req *workspacesv1.FindWorkspaceRequest) (*workspacesv1.FindWorkspaceResponse, error) {

	ws, found, err := s.database.FindWorkspace(ctx, req.GetWorkspaceId())
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	if !found {
		return nil, errors.New(errors.ErrNotFound, fmt.Errorf("workspace %s does not exist", req.GetWorkspaceId()))
	}
	return &workspacesv1.FindWorkspaceResponse{Workspace: ws}, nil
}
