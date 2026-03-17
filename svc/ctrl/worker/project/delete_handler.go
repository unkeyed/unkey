package project

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Delete removes a project by delegating all resource cleanup to each app's
// virtual object, then deleting the project record itself.
//
// Key: project_id
func (s *Service) Delete(
	ctx restate.ObjectContext,
	_ *hydrav1.DeleteProjectRequest,
) (*hydrav1.DeleteProjectResponse, error) {
	projectID := restate.Key(ctx)

	logger.Info("starting project deletion", "project_id", projectID)

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
