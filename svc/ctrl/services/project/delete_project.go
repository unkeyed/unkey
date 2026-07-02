package project

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// DeleteProject enqueues a durable Restate workflow that cascades through
// the project's apps and environments, cleaning up all associated resources.
// Returns immediately after the workflow is enqueued; actual deletion
// is eventually consistent.
//
// The project.delete audit log is written inside the workflow, not here: a
// Restate enqueue can't share a transaction with a DB write, so an after-enqueue
// insert would leave a deleting-but-unaudited window on failure. The workflow
// owns the audit write as part of its durable unit and threads the correlation
// ID down to every cascaded app.delete and environment.delete so the whole
// teardown groups under one ID.
func (s *Service) DeleteProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteProjectRequest],
) (*connect.Response[ctrlv1.DeleteProjectResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	_, err := s.db.FindProjectById(ctx, req.Msg.GetProjectId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", req.Msg.GetProjectId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find project: %w", err))
	}

	client := hydrav1.NewProjectServiceIngressClient(s.restate, req.Msg.GetProjectId())
	_, err = client.Delete().Send(ctx, &hydrav1.DeleteProjectRequest{
		Actor:         req.Msg.GetActor(),
		CorrelationId: auditlog.NewCorrelationID(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger project deletion: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteProjectResponse{}), nil
}
