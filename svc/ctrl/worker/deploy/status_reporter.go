package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// deploymentStatusReporter reports deployment progress to an external system.
// Use githubStatusReporter for GitHub-connected deployments, or noopStatusReporter
// for deployments without a GitHub connection.
type deploymentStatusReporter interface {
	Create(ctx restate.ObjectSharedContext)
	Report(ctx restate.ObjectSharedContext, state string, description string)
}

// createStatusReporter builds the appropriate deployment status reporter.
// Returns a GitHub reporter if a repo connection exists, otherwise a noop.
func (w *Workflow) createStatusReporter(
	ctx restate.ObjectContext,
	deployment db.Deployment,
	project db.Project,
	app db.App,
	environment db.Environment,
	workspace db.Workspace,
) deploymentStatusReporter {
	repoConn, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.GithubRepoConnection, error) {
		return db.Query.FindGithubRepoConnectionByAppId(runCtx, w.db.RO(), deployment.AppID)
	}, restate.WithName("find github repo connection"))
	if err != nil {
		logger.Info("no github repo connection, skipping deployment status reporting",
			"app_id", deployment.AppID,
			"error", err,
		)
		return NewNoopStatusReporter()
	}

	envLabel := project.Slug + " - " + environment.Slug
	if app.Slug != "default" {
		envLabel = project.Slug + "/" + app.Slug + " - " + environment.Slug
	}

	prefix := project.Slug
	if app.Slug != "default" {
		prefix = project.Slug + "-" + app.Slug
	}
	envURL := fmt.Sprintf("https://%s-%s-%s.%s", prefix, environment.Slug, workspace.Slug, w.defaultDomain)
	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s", w.dashboardURL, workspace.Slug, project.ID, deployment.ID)

	reporter := newGithubStatusReporter(
		w.github, w.db,
		repoConn.InstallationID, repoConn.RepositoryFullName,
		deployment.GitCommitSha.String, envLabel, envURL, logURL,
		deployment.ID, environment.Slug == "production",
	)
	reporter.Create(ctx)
	return reporter
}
