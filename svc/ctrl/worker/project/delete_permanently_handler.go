package project

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// DeletePermanently removes a project by waiting for each app's
// virtual object to complete its own permanent delete, then deleting
// the project record and the deletions row. The chain uses sequential
// synchronous Requests so by the time this handler removes the
// deletions row at the end, every descendant resource is already gone.
// In-flight deployments are cancelled inside the environment delete
// handler (the closest owner of deployment rows), which the
// app -> environment cascade fans out to.
//
// Key: project_id
func (s *Service) DeletePermanently(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteProjectPermanentlyRequest,
) (*hydrav1.DeleteProjectPermanentlyResponse, error) {
	projectID := restate.Key(ctx)

	logger.Info("starting project permanent deletion", "project_id", projectID)

	apps, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListAppIdsByProject(runCtx, s.db.RO(), projectID)
	}, restate.WithName("list apps"))
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	// Sequential synchronous Requests: each app's hard delete must finish
	// before the next app is dispatched. A failure aborts the cascade;
	// the deletions row stays intact so the cron sweep retries the whole
	// chain on its next tick.
	for _, appID := range apps {
		logger.Info("deleting app permanently", "project_id", projectID, "app_id", appID)
		if _, err := hydrav1.NewAppServiceClient(ctx, appID).
			DeletePermanently().
			Request(&hydrav1.DeleteAppPermanentlyRequest{}); err != nil {
			return nil, fmt.Errorf("app %s permanent delete: %w", appID, err)
		}
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.DeleteProjectById(runCtx, s.db.RW(), projectID)
	}, restate.WithName("delete project")); err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	// The deletions row is removed by the cron sweep once this Request
	// returns successfully. Same shape regardless of the cascade root,
	// so a project-rooted vs. app-rooted vs. env-rooted vs.
	// namespace-rooted deletion all clean up identically.

	logger.Info("project permanent deletion complete", "project_id", projectID)

	return &hydrav1.DeleteProjectPermanentlyResponse{}, nil
}
