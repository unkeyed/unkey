package project

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/resourcecleanup"
)

// The project.delete audit log is written here as a durable step rather than
// on the enqueueing RPC, so the audit record is tied to the retried unit. The
// actor and correlation ID are forwarded to each app delete (and on to each
// environment delete) so the whole teardown groups under one correlation ID.
//
// Key: project_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteProjectRequest,
) (*hydrav1.DeleteProjectResponse, error) {
	projectID := restate.Key(ctx)

	logger.Info("starting project deletion", "project_id", projectID)

	// Capture project metadata before the row is deleted, for the audit log.
	project, err := restate.Run(ctx, func(runCtx restate.RunContext) (*db.Project, error) {
		project, err := s.db.FindProjectById(runCtx, projectID)
		if db.IsNotFound(err) {
			return nil, nil
		}
		return &project, err
	}, restate.WithName("find project"))
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.MarkProjectDeleting(runCtx, db.MarkProjectDeletingParams{
			ID:         projectID,
			DeletingAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark project deleting")); err != nil {
		return nil, fmt.Errorf("mark project deleting: %w", err)
	}

	apps, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return s.db.ListAppIdsByProject(runCtx, db.ListAppIdsByProjectParams{ProjectID: projectID})
	}, restate.WithName("list apps"))
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	for _, appID := range apps {
		logger.Info("deleting app", "project_id", projectID, "app_id", appID)

		appClient := hydrav1.NewAppServiceClient(ctx, appID)
		if _, err := appClient.Delete().Request(&hydrav1.DeleteAppRequest{
			Actor:         req.GetActor(),
			CorrelationId: req.GetCorrelationId(),
		}); err != nil {
			return nil, fmt.Errorf("delete app %s: %w", appID, err)
		}
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteProjectById(runCtx, projectID)
	}, restate.WithName("delete project")); err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	if err := resourcecleanup.Schedule(ctx); err != nil {
		return nil, fmt.Errorf("schedule resource cleanup: %w", err)
	}
	if project == nil {
		return &hydrav1.DeleteProjectResponse{}, nil
	}

	if err := audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         req.GetActor(),
		CorrelationID: req.GetCorrelationId(),
		WorkspaceID:   project.WorkspaceID,
		Event:         auditlog.ProjectDeleteEvent,
		Display:       fmt.Sprintf("Deleted project %s", project.ID),
		Resource: auditlog.AuditLogResource{
			ID:          project.ID,
			Type:        auditlog.ProjectResourceType,
			Meta:        map[string]any{"name": project.Name, "slug": project.Slug},
			Name:        project.Name,
			DisplayName: project.Name,
		},
	}); err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	logger.Info("project deletion complete", "project_id", projectID)

	return &hydrav1.DeleteProjectResponse{}, nil
}
