package project

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// projectDeletedMessage is stamped onto in-flight deployment steps when a
// project is being deleted. The project (and its deployment views) are
// gone by the time anyone could look, so this is never user-visible. It
// exists for DB forensics: when someone later asks "why did this
// deployment end up cancelled?" the answer is one query away.
const projectDeletedMessage = "Project deleted"

// Delete removes a project by cancelling any in-flight deployment
// invocations, then delegating all resource cleanup to each app's virtual
// object, and finally deleting the project record itself.
//
// Cancellation runs first so the env-level cascade doesn't drop deployment
// rows out from under workflows that are still mid-build.
//
// Key: project_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteProjectRequest,
) (*hydrav1.DeleteProjectResponse, error) {
	projectID := restate.Key(ctx)

	logger.Info("starting project deletion", "project_id", projectID)

	if err := s.cancelActiveDeployments(ctx, projectID); err != nil {
		return nil, fmt.Errorf("cancel active deployments: %w", err)
	}

	apps, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListAppIdsByProject(runCtx, s.db.RO(), projectID)
	}, restate.WithName("list apps"))
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	for _, appID := range apps {
		logger.Info("deleting app", "project_id", projectID, "app_id", appID)

		appClient := hydrav1.NewAppServiceClient(ctx, appID)
		appClient.Delete().Send(&hydrav1.DeleteAppRequest{})
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteProjectById(runCtx, s.db.RW(), projectID)
	}, restate.WithName("delete project")); err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	logger.Info("project deletion complete", "project_id", projectID)

	return &hydrav1.DeleteProjectResponse{}, nil
}

// cancelActiveDeployments stamps the cancelled marker on in-flight steps,
// flips the deployments to status=cancelled, and asks the Restate admin
// API to abort each invocation. Per-step errors are non-fatal so one
// stuck invocation doesn't block the rest of the deletion.
func (s *Service) cancelActiveDeployments(ctx restate.ObjectContext, projectID string) error {
	active, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListActiveDeploymentsByProjectIdRow, error) {
		return db.Query.ListActiveDeploymentsByProjectId(runCtx, s.db.RO(), projectID)
	}, restate.WithName("list active deployments"))
	if err != nil {
		return fmt.Errorf("list active deployments: %w", err)
	}

	if len(active) == 0 {
		return nil
	}

	deploymentIDs := make([]string, 0, len(active))
	for _, d := range active {
		deploymentIDs = append(deploymentIDs, d.ID)
	}

	logger.Info("cancelling in-flight deployments for project deletion",
		"project_id", projectID,
		"count", len(deploymentIDs),
	)

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		return db.Query.EndActiveDeploymentStepsForDeployments(runCtx, s.db.RW(), db.EndActiveDeploymentStepsForDeploymentsParams{
			EndedAt:       now,
			Error:         sql.NullString{Valid: true, String: projectDeletedMessage},
			DeploymentIds: deploymentIDs,
		})
	}, restate.WithName("stamp cancelled marker on steps")); err != nil {
		logger.Warn("failed to stamp project-deleted marker on deployment steps",
			"project_id", projectID,
			"error", err,
		)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		return db.Query.UpdateDeploymentStatusBatch(runCtx, s.db.RW(), db.UpdateDeploymentStatusBatchParams{
			Status:    db.DeploymentsStatusCancelled,
			UpdatedAt: now,
			Ids:       deploymentIDs,
		})
	}, restate.WithName("mark deployments cancelled")); err != nil {
		logger.Warn("failed to batch-mark deployments cancelled",
			"project_id", projectID,
			"error", err,
		)
	}

	if s.admin == nil {
		logger.Warn("restate admin client not configured; skipping invocation cancel",
			"project_id", projectID,
		)
		return nil
	}

	for _, d := range active {
		if !d.InvocationID.Valid || d.InvocationID.String == "" {
			continue
		}

		invocationID := d.InvocationID.String
		deploymentID := d.ID

		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.admin.CancelInvocation(runCtx, invocationID)
		}, restate.WithName("cancel invocation "+deploymentID)); err != nil {
			logger.Error("failed to cancel deployment invocation",
				"project_id", projectID,
				"deployment_id", deploymentID,
				"invocation_id", invocationID,
				"error", err,
			)
		}
	}

	return nil
}
