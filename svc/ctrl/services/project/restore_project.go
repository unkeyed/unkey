package project

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// RestoreProject cancels a previously scheduled DeleteProject by firing
// the ProjectService.Restore VO chain. The VO reads
// delete_permanently_at from the project's deletions row and cascades
// the same T to descendants — only resources whose deletions row carries
// matching T get restored, so independently-deleted children are left
// alone. Deployments are not restored; the user is expected to trigger
// a fresh deployment.
//
// Returns NotFound if the project row is gone (cron sweep already
// performed the hard delete).
func (s *Service) RestoreProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.RestoreProjectRequest],
) (*connect.Response[ctrlv1.RestoreProjectResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	projectID := req.Msg.GetProjectId()

	if _, err := db.Query.FindProjectAnyById(ctx, s.db.RO(), projectID); err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", projectID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load project: %w", err))
	}

	client := hydrav1.NewProjectServiceIngressClient(s.restate, projectID)
	if _, err := client.Restore().Send(ctx, &hydrav1.RestoreProjectRequest{}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger restore: %w", err))
	}

	return connect.NewResponse(&ctrlv1.RestoreProjectResponse{}), nil
}
