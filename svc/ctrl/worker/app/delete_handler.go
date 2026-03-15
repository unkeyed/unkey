package app

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Delete removes an app by delegating environment cleanup to each environment's
// virtual object, then deleting app-level resources and the app record itself.
//
// Key: app_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteAppRequest,
) (*hydrav1.DeleteAppResponse, error) {
	appID := restate.Key(ctx)

	logger.Info("starting app deletion", "app_id", appID)

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListEnvironmentIdsByApp(runCtx, s.db.RO(), appID)
	}, restate.WithName("list environments"))
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	for _, envID := range envIDs {
		logger.Info("deleting environment", "app_id", appID, "environment_id", envID)

		envClient := hydrav1.NewEnvironmentServiceClient(ctx, envID)
		envClient.Delete().Send(&hydrav1.DeleteEnvironmentRequest{})
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

	logger.Info("app deletion complete", "app_id", appID)

	return &hydrav1.DeleteAppResponse{}, nil
}
