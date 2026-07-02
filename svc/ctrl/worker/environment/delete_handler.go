package environment

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
)

// envDeletedMessage is stamped onto in-flight deployment steps when an
// environment is being deleted. The environment (and its deployment views)
// are gone by the time anyone could look, so this is never user-visible.
const envDeletedMessage = "Environment deleted"

// Delete removes an environment and all associated resources.
//
// In-flight deployments are cancelled first so the cascade below doesn't
// drop deployment rows out from under workflows that are still mid-build.
// This handler is the single chokepoint for deployment row deletion;
// project and app deletes fan out to here via the virtual object cascade.
//
// Key: environment_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteEnvironmentRequest,
) (*hydrav1.DeleteEnvironmentResponse, error) {
	envID := restate.Key(ctx)

	logger.Info("starting environment deletion", "environment_id", envID)

	// Capture env metadata before the row is deleted, for the audit log.
	env, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return s.db.FindEnvironmentById(runCtx, envID)
	}, restate.WithName("find environment"))
	if err != nil {
		return nil, fmt.Errorf("find environment: %w", err)
	}

	if err := s.cancelProgressingDeployments(ctx, envID); err != nil {
		return nil, fmt.Errorf("cancel progressing deployments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteCiliumNetworkPoliciesByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete network policies")); err != nil {
		return nil, fmt.Errorf("delete network policies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteCustomDomainsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete custom domains")); err != nil {
		return nil, fmt.Errorf("delete custom domains: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteFrontlineRoutesByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete frontline routes")); err != nil {
		return nil, fmt.Errorf("delete frontline routes: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppEnvVarsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete env vars")); err != nil {
		return nil, fmt.Errorf("delete env vars: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppRegionalSettingsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete regional settings")); err != nil {
		return nil, fmt.Errorf("delete regional settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppBuildSettingsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete build settings")); err != nil {
		return nil, fmt.Errorf("delete build settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppRuntimeSettingsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete runtime settings")); err != nil {
		return nil, fmt.Errorf("delete runtime settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteDeploymentStepsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete deployment steps")); err != nil {
		return nil, fmt.Errorf("delete deployment steps: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteDeploymentTopologiesByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete deployment topologies")); err != nil {
		return nil, fmt.Errorf("delete deployment topologies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteDeploymentsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete deployments")); err != nil {
		return nil, fmt.Errorf("delete deployments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteEnvironmentById(runCtx, envID)
	}, restate.WithName("delete environment")); err != nil {
		return nil, fmt.Errorf("delete environment: %w", err)
	}

	// The environment has no display name, so its slug stands in.
	if err := audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         req.GetActor(),
		CorrelationID: req.GetCorrelationId(),
		WorkspaceID:   env.WorkspaceID,
		Event:         auditlog.EnvironmentDeleteEvent,
		Display:       fmt.Sprintf("Deleted environment %s", env.Slug),
		Resource: auditlog.AuditLogResource{
			ID:          env.ID,
			Type:        auditlog.EnvironmentResourceType,
			Meta:        map[string]any{"slug": env.Slug, "appId": env.AppID, "projectId": env.ProjectID},
			Name:        env.Slug,
			DisplayName: env.Slug,
		},
	}); err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	logger.Info("environment deletion complete", "environment_id", envID)

	return &hydrav1.DeleteEnvironmentResponse{}, nil
}

// cancelProgressingDeployments aborts in-flight Restate invocations, then
// marks the deployments cancelled. Cancel must happen first: if we flipped
// status up front and a CancelInvocation later failed, the retry's
// ListProgressingDeploymentsByEnvironmentId would skip the now-terminal row
// and the invocation would leak. DB errors are non-fatal since the cascade
// drops the rows anyway.
func (s *Service) cancelProgressingDeployments(ctx restate.ObjectContext, envID string) error {
	active, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListProgressingDeploymentsByEnvironmentIdRow, error) {
		return s.db.ListProgressingDeploymentsByEnvironmentId(runCtx, db.ListProgressingDeploymentsByEnvironmentIdParams{
			EnvironmentID:       envID,
			ProgressingStatuses: db.ProgressingDeploymentStatuses,
		})
	}, restate.WithName("list progressing deployments"))
	if err != nil {
		return fmt.Errorf("list progressing deployments: %w", err)
	}

	if len(active) == 0 {
		return nil
	}

	deploymentIDs := make([]string, 0, len(active))
	for _, d := range active {
		deploymentIDs = append(deploymentIDs, d.ID)
	}

	logger.Info("cancelling in-flight deployments for environment deletion",
		"environment_id", envID,
		"count", len(deploymentIDs),
	)

	// DB errors below are non-fatal: even if we fail to flip the row state
	// here, the cascade below (DeleteDeploymentsByEnvironmentId) drops the
	// row anyway. We continue so the Restate-side cancel still fires.
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		return s.db.EndActiveDeploymentStepsForDeployments(runCtx, db.EndActiveDeploymentStepsForDeploymentsParams{
			EndedAt:       now,
			Error:         sql.NullString{Valid: true, String: envDeletedMessage},
			DeploymentIds: deploymentIDs,
		})
	}, restate.WithName("stamp cancelled marker on steps")); err != nil {
		logger.Warn("failed to stamp env-deleted marker on deployment steps",
			"environment_id", envID,
			"error", err,
		)
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
			return fmt.Errorf("cancel invocation %s for deployment %s: %w", invocationID, deploymentID, err)
		}
	}

	// Status flip is best-effort: invocations are already cancelled, and the
	// cascade below drops these rows entirely. Propagating would deadlock the
	// handler because Restate journals the error and replays it on every retry.
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		return s.db.UpdateDeploymentStatusBatch(runCtx, db.UpdateDeploymentStatusBatchParams{
			Status:    db.DeploymentsStatusCancelled,
			UpdatedAt: now,
			Ids:       deploymentIDs,
		})
	}, restate.WithName("mark deployments cancelled")); err != nil {
		logger.Warn("failed to batch-mark deployments cancelled",
			"environment_id", envID,
			"error", err,
		)
	}

	return nil
}
