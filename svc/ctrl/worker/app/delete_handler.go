package app

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
		row, findErr := s.db.FindAppById(runCtx, appID)
		return row, restateutil.TerminalNotFound(findErr, db.IsNotFound)
	}, restate.WithName("find app"))
	if err != nil {
		if !restateutil.IsNotFoundTerminal(err) {
			return nil, fmt.Errorf("find app: %w", err)
		}
		// Another delete already removed the row, so the postcondition holds
		logger.Info("app already deleted", "app_id", appID)
		return &hydrav1.DeleteAppResponse{}, nil
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
		envClient.Delete().Send(&hydrav1.DeleteEnvironmentRequest{
			Actor:         req.GetActor(),
			CorrelationId: req.GetCorrelationId(),
		})
	}

	// The environment deletes above are fire-and-forget and only reach domains
	// whose environment they know about. A leftover domain row blocks the
	// customer from attaching the hostname to another app, so remove the app's
	// domains and their routes here as well.
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteCustomDomainsByAppId(runCtx, appID)
	}, restate.WithName("delete custom domains")); err != nil {
		return nil, fmt.Errorf("delete custom domains: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteFrontlineRoutesByAppId(runCtx, appID)
	}, restate.WithName("delete frontline routes")); err != nil {
		return nil, fmt.Errorf("delete frontline routes: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteGithubRepoConnectionsByAppId(runCtx, appID)
	}, restate.WithName("delete github repo connections")); err != nil {
		return nil, fmt.Errorf("delete github repo connections: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppSourceOciByAppId(runCtx, appID)
	}, restate.WithName("delete OCI source")); err != nil {
		return nil, fmt.Errorf("delete OCI source: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppById(runCtx, appID)
	}, restate.WithName("delete app")); err != nil {
		return nil, fmt.Errorf("delete app: %w", err)
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
