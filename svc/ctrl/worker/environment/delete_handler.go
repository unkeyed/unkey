package environment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
		return s.terminateSubscriptionsAndDeleteSentinels(runCtx, envID)
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

// terminateSubscriptionsAndDeleteSentinels closes every open billing
// subscription for sentinels in this environment, then hard-deletes the
// sentinel rows. Terminating first is required so the subscription
// interval log has a clean `terminated_at` at the moment of deletion —
// otherwise billing would read the open row as "still running forever."
//
// Krane learns about the deletion via its 60s desired-state resync loop:
// `GetDesiredSentinelState` returns CodeNotFound for the deleted sentinel,
// and krane tears down the k8s Deployment. No incremental-watch Delete
// event is emitted because `deployment_changes` doesn't carry enough info
// for the watch handler to resolve a hard-deleted resource (a follow-up
// PR adds `op` + `k8s_name` + `k8s_namespace` columns to close that gap
// for all resource types).
func (s *Service) terminateSubscriptionsAndDeleteSentinels(ctx context.Context, envID string) error {
	return db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if err := db.Query.TerminateOpenSentinelSubscriptionsByEnvironment(txCtx, tx, db.TerminateOpenSentinelSubscriptionsByEnvironmentParams{
			TerminatedAt:  sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			EnvironmentID: envID,
		}); err != nil {
			return fmt.Errorf("terminate sentinel subscriptions: %w", err)
		}
		return db.Query.DeleteSentinelsByEnvironmentId(txCtx, tx, envID)
	})
}
