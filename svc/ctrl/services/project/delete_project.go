package project

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
// Returns after marking the project deleted and notifying its deployment topologies;
// physical deletion is eventually consistent.
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

	project, err := s.db.FindProjectById(ctx, req.Msg.GetProjectId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", req.Msg.GetProjectId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find project: %w", err))
	}

	if project.DeletedAt.Valid {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", project.ID))
	}

	client := hydrav1.NewProjectServiceIngressClient(s.restate, req.Msg.GetProjectId())
	_, err = client.Delete().Send(ctx, &hydrav1.DeleteProjectRequest{
		Actor:         req.Msg.GetActor(),
		CorrelationId: auditlog.NewCorrelationID(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger project deletion: %w", err))
	}

	err = db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		now := time.Now().UnixMilli()
		if err := queries.MarkProjectDeleted(ctx, db.MarkProjectDeletedParams{
			ID: project.ID, DeletedAt: sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return err
		}
		return queries.NotifyDeploymentTopologiesByProject(ctx, db.NotifyDeploymentTopologiesByProjectParams{
			ProjectID: project.ID, UpdatedAt: now,
		})
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to mark project deleted: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteProjectResponse{}), nil
}
