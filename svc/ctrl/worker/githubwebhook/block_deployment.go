package githubwebhook

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// blockDeploymentForApproval creates a GitHub commit status to signal that
// the push requires authorization from a project member. Clicking "Details"
// in the PR goes directly to the dashboard authorize page.
func (s *Service) blockDeploymentForApproval(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	project db.Project,
	repo db.GithubRepoConnection,
	deploymentID string,
) error {
	workspace, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Workspace, error) {
		return db.Query.FindWorkspaceByID(runCtx, s.db.RO(), project.WorkspaceID)
	}, restate.WithName("find workspace for approval log url"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		return err
	}

	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s",
		s.dashboardURL, workspace.Slug, project.ID, deploymentID,
	)

	if !s.allowUnauthenticatedDeployments {
		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return s.github.CreateCommitStatus(
				repo.InstallationID,
				req.GetRepositoryFullName(),
				req.GetAfter(),
				"failure",
				logURL,
				"Awaiting authorization from a project member",
				"Unkey Deploy Authorization",
			)
		}, restate.WithName("create commit status for authorization"), restate.WithMaxRetryDuration(30*time.Second))
	}

	logger.Info("deployment blocked for authorization",
		"deployment_id", deploymentID,
		"project_id", project.ID,
		"sender", req.GetSenderLogin(),
	)

	return nil
}
