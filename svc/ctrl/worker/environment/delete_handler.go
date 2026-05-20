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
	_ *hydrav1.DeleteEnvironmentRequest,
) (*hydrav1.DeleteEnvironmentResponse, error) {
	envID := restate.Key(ctx)

	logger.Info("starting environment deletion", "environment_id", envID)

	if err := s.cancelActiveDeployments(ctx, envID); err != nil {
		return nil, fmt.Errorf("cancel active deployments: %w", err)
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

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteFrontlineRoutesByEnvironmentId(runCtx, s.db.RW(), envID)
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

	logger.Info("environment deletion complete", "environment_id", envID)

	return &hydrav1.DeleteEnvironmentResponse{}, nil
}

// cancelActiveDeployments stamps the cancelled marker on in-flight steps,
// flips the deployments to status=cancelled, and asks the Restate admin
// API to abort each invocation. Per-step errors are non-fatal so one
// stuck invocation doesn't block the rest of the deletion.
func (s *Service) cancelActiveDeployments(ctx restate.ObjectContext, envID string) error {
	active, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListActiveDeploymentsByEnvironmentIdRow, error) {
		return db.Query.ListActiveDeploymentsByEnvironmentId(runCtx, s.db.RO(), db.ListActiveDeploymentsByEnvironmentIdParams{
			EnvironmentID:    envID,
			TerminalStatuses: db.TerminalDeploymentStatuses,
		})
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
				"environment_id", envID,
				"deployment_id", deploymentID,
				"invocation_id", invocationID,
				"error", err,
			)
		}
	}

	return nil
}
