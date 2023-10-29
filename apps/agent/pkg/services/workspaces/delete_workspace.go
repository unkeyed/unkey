package workspaces

import (
	"context"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

func (s *service) DeleteWorkspace(ctx context.Context, req *workspacesv1.DeleteWorkspaceRequest) (*workspacesv1.DeleteWorkspaceResponse, error) {

	err := s.database.DeleteWorkspace(ctx, req.GetWorkspaceId())
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	return &workspacesv1.DeleteWorkspaceResponse{}, nil
}
