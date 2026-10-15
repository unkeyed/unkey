package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Key: app_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteAppRequest,
) (*hydrav1.DeleteAppResponse, error) {
	appID := restate.Key(ctx)

	logger.Info("starting app deletion", "app_id", appID)

	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (*db.App, error) {
		app, err := s.db.FindAppById(runCtx, appID)
		if db.IsNotFound(err) {
			return nil, nil
		}
		return &app, err
	}, restate.WithName("find app"))
	if err != nil {
		return nil, fmt.Errorf("find app: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.TxRetry(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			queries := db.NewQueries(tx)
			now := time.Now().UnixMilli()
			if err := queries.MarkAppDeleted(txCtx, db.MarkAppDeletedParams{
				ID: appID, DeletedAt: sql.NullInt64{Valid: true, Int64: now},
			}); err != nil {
				return err
			}
			return queries.NotifyDeploymentTopologiesByApp(txCtx, db.NotifyDeploymentTopologiesByAppParams{
				AppID: appID, UpdatedAt: now,
			})
		})
	}, restate.WithName("mark app deleting")); err != nil {
		return nil, fmt.Errorf("mark app deleting: %w", err)
	}

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return s.db.ListEnvironmentIdsByApp(runCtx, db.ListEnvironmentIdsByAppParams{AppID: appID})
	}, restate.WithName("list environments"))
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	for _, envID := range envIDs {
		logger.Info("deleting environment", "app_id", appID, "environment_id", envID)

		envClient := hydrav1.NewEnvironmentServiceClient(ctx, envID)
		if _, err := envClient.Delete().Request(&hydrav1.DeleteEnvironmentRequest{
			Actor:         req.GetActor(),
			CorrelationId: req.GetCorrelationId(),
		}); err != nil {
			return nil, fmt.Errorf("delete environment %s: %w", envID, err)
		}
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppById(runCtx, appID)
	}, restate.WithName("delete app")); err != nil {
		return nil, fmt.Errorf("delete app: %w", err)
	}

	if app == nil {
		return &hydrav1.DeleteAppResponse{}, nil
	}

	if err := audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         req.GetActor(),
		CorrelationID: req.GetCorrelationId(),
		WorkspaceID:   app.WorkspaceID,
		Event:         auditlog.AppDeleteEvent,
		Display:       fmt.Sprintf("Deleted app %s", app.ID),
		Targets: []auditlog.AuditLogTarget{{
			ID:          app.ID,
			Type:        auditlog.AppTargetType,
			Meta:        map[string]any{"name": app.Name, "slug": app.Slug, "projectId": app.ProjectID},
			Name:        app.Name,
			DisplayName: app.Name,
		}},
	}); err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	logger.Info("app deletion complete", "app_id", appID)

	return &hydrav1.DeleteAppResponse{}, nil
}
