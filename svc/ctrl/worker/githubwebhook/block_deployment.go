package githubwebhook

import (
	"fmt"
	"net/url"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// blockDeploymentForApproval creates GitHub Deployment and Check Run entries
// to signal that the push requires authorization from a project member.
func (s *Service) blockDeploymentForApproval(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	project db.Project,
	env db.Environment,
	app db.App,
	repo db.GithubRepoConnection,
	branch string,
) error {
	envLabel := project.Slug + " - " + env.Slug
	if app.Slug != "default" {
		envLabel = project.Slug + "/" + app.Slug + " - " + env.Slug
	}

	workspace, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Workspace, error) {
		return db.Query.FindWorkspaceByID(runCtx, s.db.RO(), project.WorkspaceID)
	}, restate.WithName("find workspace for approval log url"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		return err
	}

	logURL := fmt.Sprintf("%s/%s/projects/%s/authorize?branch=%s&sha=%s&sender=%s&message=%s",
		s.dashboardURL, workspace.Slug, project.ID,
		url.QueryEscape(branch),
		url.QueryEscape(req.GetAfter()),
		url.QueryEscape(req.GetSenderLogin()),
		url.QueryEscape(req.GetCommitMessage()),
	)

	ghDeploymentID, ghErr := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return s.github.CreateDeployment(
			repo.InstallationID,
			req.GetRepositoryFullName(),
			req.GetAfter(),
			envLabel,
			"Awaiting authorization",
			env.Slug == "production",
		)
	}, restate.WithName("create github deployment for approval"), restate.WithMaxRetryDuration(30*time.Second))
	if ghErr != nil {
		logger.Error("failed to create GitHub deployment for approval", "error", ghErr,
			"project_id", project.ID, "app_id", app.ID)
	} else {
		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return s.github.CreateDeploymentStatus(
				repo.InstallationID,
				req.GetRepositoryFullName(),
				ghDeploymentID,
				"failure",
				"",
				logURL,
				"Awaiting authorization from a project member",
			)
		}, restate.WithName("github deployment status: failure (awaiting auth)"), restate.WithMaxRetryDuration(30*time.Second))
	}

	_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
		_, crErr := s.github.CreateCheckRun(
			repo.InstallationID,
			req.GetRepositoryFullName(),
			req.GetAfter(),
			"Unkey Deployment Authorization",
			"completed",
			"action_required",
			"", // no output title — clicking the check run redirects directly to details_url
			"",
			logURL,
		)
		return crErr
	}, restate.WithName("create check run for authorization"), restate.WithMaxRetryDuration(30*time.Second))

	logger.Info("deployment blocked for authorization",
		"project_id", project.ID,
		"app_id", app.ID,
		"branch", branch,
		"sender", req.GetSenderLogin(),
	)

	return nil
}
