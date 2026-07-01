package app

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
)

// Delete removes an app by delegating environment cleanup to each environment's
// virtual object, then deleting app-level resources and the app record itself.
//
// The app.delete audit log is written here as a durable step rather than on the
// RPC that enqueued this workflow: a Restate enqueue can't share a transaction
// with the audit insert, so writing it on the RPC path risked a
// deleting-but-unaudited window. The actor and correlation ID are threaded in
// via the request and forwarded to each environment delete so the whole teardown
// groups under one correlation ID.
//
// Key: app_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteAppRequest,
) (*hydrav1.DeleteAppResponse, error) {
	appID := restate.Key(ctx)

	logger.Info("starting app deletion", "app_id", appID)

	// Capture the app's metadata before the cascade deletes the row, so the
	// audit log written at the end still has a name/slug to display.
	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
		return db.Query.FindAppById(runCtx, s.db.RO(), appID)
	}, restate.WithName("find app"))
	if err != nil {
		return nil, fmt.Errorf("find app: %w", err)
	}

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListEnvironmentIdsByApp(runCtx, s.db.RO(), appID)
	}, restate.WithName("list environments"))
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	for _, envID := range envIDs {
		logger.Info("deleting environment", "app_id", appID, "environment_id", envID)

		envClient := hydrav1.NewEnvironmentServiceClient(ctx, envID)
		envClient.Delete().Send(&hydrav1.DeleteEnvironmentRequest{
			Actor:         req.GetActor(),
			CorrelationId: req.GetCorrelationId(),
		})
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteGithubRepoConnectionsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete github repo connections")); err != nil {
		return nil, fmt.Errorf("delete github repo connections: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppById(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete app")); err != nil {
		return nil, fmt.Errorf("delete app: %w", err)
	}

	if err := audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         req.GetActor(),
		CorrelationID: req.GetCorrelationId(),
		WorkspaceID:   app.WorkspaceID,
		Event:         auditlog.AppDeleteEvent,
		Display:       fmt.Sprintf("Deleted app %s", app.ID),
		Resource: auditlog.AuditLogResource{
			ID:          app.ID,
			Type:        auditlog.AppResourceType,
			Meta:        map[string]any{"name": app.Name, "slug": app.Slug, "projectId": app.ProjectID},
			Name:        app.Name,
			DisplayName: app.Name,
		},
	}); err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	logger.Info("app deletion complete", "app_id", appID)

	return &hydrav1.DeleteAppResponse{}, nil
}
