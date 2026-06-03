package environment

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// envDeletedMessage is stamped onto in-flight deployment steps when an
// environment is being deleted. The environment (and its deployment views)
// are gone by the time anyone could look, so this is never user-visible.
const envDeletedMessage = "Environment deleted"

// DeletePermanently removes an environment and all associated resources.
//
// In-flight deployments are cancelled first so the cascade below doesn't
// drop deployment rows out from under workflows that are still mid-build.
// This handler is the single chokepoint for deployment row deletion;
// project and app permanent-deletes fan out to here via the virtual
// object cascade.
//
// Key: environment_id
func (s *Service) DeletePermanently(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteEnvironmentPermanentlyRequest,
) (*hydrav1.DeleteEnvironmentPermanentlyResponse, error) {
	envID := restate.Key(ctx)

	logger.Info("starting environment permanent deletion", "environment_id", envID)

	if err := s.cancelProgressingDeployments(ctx, envID); err != nil {
		return nil, fmt.Errorf("cancel progressing deployments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteCiliumNetworkPoliciesByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete network policies")); err != nil {
		return nil, fmt.Errorf("delete network policies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteSentinelsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete sentinels")); err != nil {
		return nil, fmt.Errorf("delete sentinels: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteCustomDomainsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete custom domains")); err != nil {
		return nil, fmt.Errorf("delete custom domains: %w", err)
	}

	// Paginated to bound transaction size for environments with many
	// routes; loop continues until a partial page comes back, which
	// guarantees the table is empty for this env. Every row has
	// environment_id NOT NULL so this catches all sticky variants.
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.deleteFrontlineRoutes(runCtx, envID)
	}, restate.WithName("delete frontline routes")); err != nil {
		return nil, fmt.Errorf("delete frontline routes: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppEnvVarsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete env vars")); err != nil {
		return nil, fmt.Errorf("delete env vars: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppRegionalSettingsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete regional settings")); err != nil {
		return nil, fmt.Errorf("delete regional settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppBuildSettingsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete build settings")); err != nil {
		return nil, fmt.Errorf("delete build settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppRuntimeSettingsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete runtime settings")); err != nil {
		return nil, fmt.Errorf("delete runtime settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteDeploymentStepsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete deployment steps")); err != nil {
		return nil, fmt.Errorf("delete deployment steps: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteDeploymentTopologiesByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete deployment topologies")); err != nil {
		return nil, fmt.Errorf("delete deployment topologies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteDeploymentsByEnvironmentId(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete deployments")); err != nil {
		return nil, fmt.Errorf("delete deployments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteEnvironmentById(runCtx, s.db.RW(), envID)
	}, restate.WithName("delete environment")); err != nil {
		return nil, fmt.Errorf("delete environment: %w", err)
	}

	// The deletions row is owned by the cascade root and removed by
	// the cron handler after this Request returns. Nothing to clean
	// up at this level.

	logger.Info("environment deletion complete", "environment_id", envID)

	return &hydrav1.DeleteEnvironmentPermanentlyResponse{}, nil
}

// frontlineRouteBatchLimit caps how many rows a single DELETE round
// takes. Bounded transaction size protects against replication lag /
// row-lock blowups for environments that accumulated many routes.
const frontlineRouteBatchLimit = 1000

// deleteFrontlineRoutes deletes every frontline route for envID in
// bounded batches. Loops until a partial page comes back, which is the
// signal that the table is empty for this env id (the WHERE clause
// matched fewer than batchLimit rows).
func (s *Service) deleteFrontlineRoutes(ctx restate.RunContext, envID string) error {
	for {
		res, err := db.Query.DeleteFrontlineRoutesByEnvironmentId(ctx, s.db.RW(), db.DeleteFrontlineRoutesByEnvironmentIdParams{
			EnvironmentID: envID,
			Limit:         frontlineRouteBatchLimit,
		})
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected < frontlineRouteBatchLimit {
			return nil
		}
	}
}

// cancelProgressingDeployments aborts in-flight Restate invocations, then
// marks the deployments cancelled. Cancel must happen first: if we flipped
// status up front and a CancelInvocation later failed, the retry's
// ListProgressingDeploymentsByEnvironmentId would skip the now-terminal row
// and the invocation would leak. DB errors are non-fatal since the cascade
// drops the rows anyway.
func (s *Service) cancelProgressingDeployments(ctx restate.ObjectContext, envID string) error {
	active, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListProgressingDeploymentsByEnvironmentIdRow, error) {
		return db.Query.ListProgressingDeploymentsByEnvironmentId(runCtx, s.db.RO(), db.ListProgressingDeploymentsByEnvironmentIdParams{
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
		return db.Query.EndActiveDeploymentStepsForDeployments(runCtx, s.db.RW(), db.EndActiveDeploymentStepsForDeploymentsParams{
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
		return db.Query.UpdateDeploymentStatusBatch(runCtx, s.db.RW(), db.UpdateDeploymentStatusBatchParams{
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
