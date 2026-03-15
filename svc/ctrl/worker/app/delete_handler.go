package app

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Delete removes an app and all associated resources.
//
// Key: app_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteAppRequest,
) (*hydrav1.DeleteAppResponse, error) {
	appID := restate.Key(ctx)

	logger.Info("starting app deletion", "app_id", appID)

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteCiliumNetworkPoliciesByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete network policies")); err != nil {
		return nil, fmt.Errorf("delete network policies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteInstancesByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete instances")); err != nil {
		return nil, fmt.Errorf("delete instances: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteSentinelsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete sentinels")); err != nil {
		return nil, fmt.Errorf("delete sentinels: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteCustomDomainsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete custom domains")); err != nil {
		return nil, fmt.Errorf("delete custom domains: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteFrontlineRoutesByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete frontline routes")); err != nil {
		return nil, fmt.Errorf("delete frontline routes: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteGithubRepoConnectionsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete github repo connections")); err != nil {
		return nil, fmt.Errorf("delete github repo connections: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppEnvVarsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete env vars")); err != nil {
		return nil, fmt.Errorf("delete env vars: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppRegionalSettingsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete regional settings")); err != nil {
		return nil, fmt.Errorf("delete regional settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppBuildSettingsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete build settings")); err != nil {
		return nil, fmt.Errorf("delete build settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppRuntimeSettingsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete runtime settings")); err != nil {
		return nil, fmt.Errorf("delete runtime settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteDeploymentStepsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete deployment steps")); err != nil {
		return nil, fmt.Errorf("delete deployment steps: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteDeploymentTopologiesByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete deployment topologies")); err != nil {
		return nil, fmt.Errorf("delete deployment topologies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteDeploymentsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete deployments")); err != nil {
		return nil, fmt.Errorf("delete deployments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteEnvironmentsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete environments")); err != nil {
		return nil, fmt.Errorf("delete environments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppById(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete app")); err != nil {
		return nil, fmt.Errorf("delete app: %w", err)
	}

	logger.Info("app deletion complete", "app_id", appID)

	return &hydrav1.DeleteAppResponse{}, nil
}
