package environment

import (
	"database/sql"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Delete removes an environment and all associated resources.
//
// Key: environment_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteEnvironmentRequest,
) (*hydrav1.DeleteEnvironmentResponse, error) {
	envID := restate.Key(ctx)

	logger.Info("starting environment deletion", "environment_id", envID)

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
		return db.Query.DeleteFrontlineRoutesByEnvironmentId(runCtx, s.db.RW(), sql.NullString{Valid: true, String: envID})
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
