package app

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// DeletePermanently removes the app by waiting for each environment's
// permanent delete to complete before deleting app-level resources and
// the app row itself. Sequential synchronous Requests keep the cascade
// ordered: when this handler returns, every descendant is gone.
//
// The deletions row is owned by the cascade root (project) — this
// handler does not touch it.
//
// Key: app_id
func (s *Service) DeletePermanently(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteAppPermanentlyRequest,
) (*hydrav1.DeleteAppPermanentlyResponse, error) {
	appID := restate.Key(ctx)

	logger.Info("starting app permanent deletion", "app_id", appID)

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListEnvironmentIdsByApp(runCtx, s.db.RO(), appID)
	}, restate.WithName("list environments"))
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	for _, envID := range envIDs {
		logger.Info("deleting environment permanently", "app_id", appID, "environment_id", envID)
		if _, err := hydrav1.NewEnvironmentServiceClient(ctx, envID).
			DeletePermanently().
			Request(&hydrav1.DeleteEnvironmentPermanentlyRequest{}); err != nil {
			return nil, fmt.Errorf("environment %s permanent delete: %w", envID, err)
		}
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteGithubRepoConnectionsByAppId(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete github repo connections")); err != nil {
		return nil, fmt.Errorf("delete github repo connections: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteAppById(runCtx, s.db.RW(), appID)
	}, restate.WithName("delete app")); err != nil {
		return nil, fmt.Errorf("delete app: %w", err)
	}

	logger.Info("app permanent deletion complete", "app_id", appID)

	return &hydrav1.DeleteAppPermanentlyResponse{}, nil
}
