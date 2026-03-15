package project

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
)

// DeleteProject triggers the Restate workflow to durably clean up all
// resources associated with a project.
func (s *Service) DeleteProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteProjectRequest],
) (*connect.Response[ctrlv1.DeleteProjectResponse], error) {
	if err := assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	client := hydrav1.NewProjectServiceIngressClient(s.restate, req.Msg.GetProjectId())
	_, err := client.Delete().Send(ctx, &hydrav1.DeleteProjectRequest{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger project deletion: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteProjectResponse{}), nil
}
