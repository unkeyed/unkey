package project

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

// Delete removes a project by delegating all resource cleanup to each
// app's virtual object, then deleting the project record itself. In-flight
// deployments are cancelled inside the environment delete handler (the
// closest owner of deployment rows), which the app -> environment cascade
// fans out to.
//
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
	project, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Project, error) {
		row, findErr := s.db.FindProjectById(runCtx, projectID)
		return row, restateutil.TerminalNotFound(findErr)
	}, restate.WithName("find project"))
	if err != nil {
		if !restateutil.IsNotFoundTerminal(err) {
			return nil, fmt.Errorf("find project: %w", err)
		}
		// The row is already gone, so a repeat delete has nothing left to do
		logger.Info("project already deleted", "project_id", projectID)
		return &hydrav1.DeleteProjectResponse{}, nil
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
		appClient.Delete().Send(&hydrav1.DeleteAppRequest{
			Actor:         req.GetActor(),
			CorrelationId: req.GetCorrelationId(),
		})
	}

	// Audit before the row delete: once the row is gone a repeat delete returns
	// early, so an audit still owed by a killed earlier delete would never be
	// written
	if err := audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         req.GetActor(),
		CorrelationID: req.GetCorrelationId(),
		WorkspaceID:   project.WorkspaceID,
		Event:         auditlog.ProjectDeleteEvent,
		Display:       fmt.Sprintf("Deleted project %s", project.ID),
		Targets: []auditlog.AuditLogTarget{{
			ID:          project.ID,
			Type:        auditlog.ProjectTargetType,
			Meta:        map[string]any{"name": project.Name, "slug": project.Slug},
			Name:        project.Name,
			DisplayName: project.Name,
		}},
	}); err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteProjectById(runCtx, projectID)
	}, restate.WithName("delete project")); err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	logger.Info("project deletion complete", "project_id", projectID)

	return &hydrav1.DeleteProjectResponse{}, nil
}
