package project

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
)

func (s *Service) CreateProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateProjectRequest],
) (*connect.Response[ctrlv1.CreateProjectResponse], error) {

	res, err := hydrav1.NewProjectServiceIngressClient(s.restate, req.Msg.GetWorkspaceId()).
		CreateProject().Request(ctx, &hydrav1.CreateProjectRequest{
		WorkspaceId:   req.Msg.GetWorkspaceId(),
		Name:          req.Msg.GetName(),
		Slug:          req.Msg.GetSlug(),
		GitRepository: req.Msg.GetGitRepository(),
	})

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.CreateProjectResponse{
		Id: res.ProjectId,
	}), nil
}
