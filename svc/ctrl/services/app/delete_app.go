package app

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

// DeleteApp enqueues a durable Restate workflow that cascades through
// the app's environments and cleans up all associated resources.
// Returns after marking the app deleted and notifying its deployment topologies;
// physical deletion is eventually consistent.
func (s *Service) DeleteApp(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteAppRequest],
) (*connect.Response[ctrlv1.DeleteAppResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetAppId(), "app_id is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	app, err := s.db.FindAppById(ctx, req.Msg.GetAppId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found: %s", req.Msg.GetAppId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find app: %w", err))
	}

	if app.DeletedAt.Valid {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found: %s", app.ID))
	}
	project, err := s.db.FindProjectById(ctx, app.ProjectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found: %s", app.ID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find project: %w", err))
	}
	if project.DeletedAt.Valid {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found: %s", app.ID))
	}

	client := hydrav1.NewAppServiceIngressClient(s.restate, req.Msg.GetAppId())
	_, err = client.Delete().Send(ctx, &hydrav1.DeleteAppRequest{
		Actor:         req.Msg.GetActor(),
		CorrelationId: auditlog.NewCorrelationID(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger app deletion: %w", err))
	}

	err = db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		now := time.Now().UnixMilli()
		if err := queries.MarkAppDeleted(ctx, db.MarkAppDeletedParams{
			ID: app.ID, DeletedAt: sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return err
		}
		return queries.NotifyDeploymentTopologiesByApp(ctx, db.NotifyDeploymentTopologiesByAppParams{
			AppID: app.ID, UpdatedAt: now,
		})
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to mark app deleted: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteAppResponse{}), nil
}
