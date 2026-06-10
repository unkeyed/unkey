package project

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// DeleteProject schedules a project for permanent deletion. Mints the
// deletion_id here so the same id flows through the entire cascade,
// then fires the ProjectService.MarkForDeletion VO which inserts the
// deletions row, points the project at it, and cascades through apps
// and environments — every layer points at the same id. The
// environment step also flips deployments to status='stopped' so
// krane scales their pods to zero. The cron sweep performs the
// permanent-delete cascade once the grace window elapses.
//
// If the project is already scheduled, returns the existing
// delete_permanently_at without re-firing the VO.
func (s *Service) DeleteProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteProjectRequest],
) (*connect.Response[ctrlv1.DeleteProjectResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	projectID := req.Msg.GetProjectId()

	row, err := db.Query.FindProjectAnyById(ctx, s.db.RO(), projectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", projectID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load project: %w", err))
	}

	if row.DeletionID.Valid {
		// Already scheduled. Look up the existing T from the deletions
		// row so the response carries the truth, not a new guess.
		deletion, err := db.Query.FindDeletionById(ctx, s.db.RO(), row.DeletionID.String)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deletion row: %w", err))
		}
		return connect.NewResponse(&ctrlv1.DeleteProjectResponse{
			DeletePermanentlyAt: deletion.DeletePermanentlyAt,
		}), nil
	}

	deletionID := string(uid.New(uid.DeletionPrefix))
	deletePermanentlyAt := s.clock.Now().Add(gracePeriod).UnixMilli()

	client := hydrav1.NewProjectServiceIngressClient(s.restate, projectID)
	if _, err := client.MarkForDeletion().Send(ctx, &hydrav1.MarkProjectForDeletionRequest{
		DeletionId:          deletionID,
		DeletePermanentlyAt: deletePermanentlyAt,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger mark for deletion: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteProjectResponse{
		DeletePermanentlyAt: deletePermanentlyAt,
	}), nil
}
