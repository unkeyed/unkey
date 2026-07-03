package app

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// DeletePermanently removes the app by waiting for each environment's
// permanent delete to complete before deleting app-level resources and
// the app row itself. Sequential synchronous Requests keep the cascade
// ordered: when this handler returns, every descendant is gone.
//
// The deletions row is owned by the cascade root (project) — this
// handler does not touch it.
//
// The app.delete audit log is written here as a durable step rather than on the
// RPC that enqueued this workflow: a Restate enqueue can't share a transaction
// with the audit insert, so writing it on the RPC path risked a
// deleting-but-unaudited window. The actor and correlation ID are threaded in
// via the request and forwarded to each environment delete so the whole teardown
// groups under one correlation ID.
//
// Key: app_id
func (s *Service) DeletePermanently(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteAppPermanentlyRequest,
) (*hydrav1.DeleteAppPermanentlyResponse, error) {
	appID := restate.Key(ctx)

	logger.Info("starting app permanent deletion", "app_id", appID)

	// Capture the app's metadata before the cascade deletes the row, so the
	// audit log written at the end still has a name/slug to display.
	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
		return s.db.FindAppById(runCtx, appID)
	}, restate.WithName("find app"))
	if err != nil {
		return nil, fmt.Errorf("find app: %w", err)
	}

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return s.db.ListEnvironmentIdsByApp(runCtx, appID)
	}, restate.WithName("list environments"))
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	for _, envID := range envIDs {
		logger.Info("deleting environment permanently", "app_id", appID, "environment_id", envID)
		if _, err := hydrav1.NewEnvironmentServiceClient(ctx, envID).
			DeletePermanently().
			Request(&hydrav1.DeleteEnvironmentPermanentlyRequest{
				Actor:         req.GetActor(),
				CorrelationId: req.GetCorrelationId(),
			}); err != nil {
			return nil, fmt.Errorf("environment %s permanent delete: %w", envID, err)
		}
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteGithubRepoConnectionsByAppId(runCtx, appID)
	}, restate.WithName("delete github repo connections")); err != nil {
		return nil, fmt.Errorf("delete github repo connections: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppById(runCtx, appID)
	}, restate.WithName("delete app")); err != nil {
		return nil, fmt.Errorf("delete app: %w", err)
	}

	if req.GetCorrelationId() != "" {
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
	}

	logger.Info("app permanent deletion complete", "app_id", appID)

	return &hydrav1.DeleteAppPermanentlyResponse{}, nil
}
