package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// githubStatusReporter reports deployment status via the GitHub Deployments API.
// All API calls are fire-and-forget — GitHub being down must not block deploys.
type githubStatusReporter struct {
	github           githubclient.GitHubClient
	db               db.Database
	installationID   int64
	repo             string
	commitSHA        string
	environmentLabel string
	environmentURL   string
	logURL           string
	deploymentID     string
	isProduction     bool

	ghDeploymentID int64 // set after Create
}

// githubStatusReporterConfig holds the parameters for creating a githubStatusReporter.
type githubStatusReporterConfig struct {
	GitHub           githubclient.GitHubClient
	DB               db.Database
	InstallationID   int64
	Repo             string
	CommitSHA        string
	EnvironmentLabel string
	EnvironmentURL   string
	LogURL           string
	DeploymentID     string
	IsProduction     bool
}

func newGithubStatusReporter(cfg githubStatusReporterConfig) *githubStatusReporter {
	return &githubStatusReporter{
		github:           cfg.GitHub,
		db:               cfg.DB,
		installationID:   cfg.InstallationID,
		repo:             cfg.Repo,
		commitSHA:        cfg.CommitSHA,
		environmentLabel: cfg.EnvironmentLabel,
		environmentURL:   cfg.EnvironmentURL,
		logURL:           cfg.LogURL,
		deploymentID:     cfg.DeploymentID,
		isProduction:     cfg.IsProduction,
		ghDeploymentID:   0,
	}
}

func (r *githubStatusReporter) Create(ctx restate.ObjectSharedContext) {
	if r.installationID == 0 || r.repo == "" || r.commitSHA == "" {
		return
	}

	ghDeploymentID, err := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return r.github.CreateDeployment(
			r.installationID, r.repo, r.commitSHA,
			r.environmentLabel, "Deploying...", r.isProduction,
		)
	}, restate.WithName("create github deployment"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Error("failed to create GitHub deployment", "error", err, "deployment_id", r.deploymentID)
		return
	}

	r.ghDeploymentID = ghDeploymentID

	_ = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentGithubDeploymentId(runCtx, r.db.RW(), db.UpdateDeploymentGithubDeploymentIdParams{
			GithubDeploymentID: sql.NullInt64{Valid: true, Int64: ghDeploymentID},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:                 r.deploymentID,
		})
	}, restate.WithName("persist github deployment id"), restate.WithMaxRetryDuration(30*time.Second))

	r.Report(ctx, "pending", "Deployment queued")
}

func (r *githubStatusReporter) Report(ctx restate.ObjectSharedContext, state string, description string) {
	if r.ghDeploymentID == 0 {
		return
	}

	_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return r.github.CreateDeploymentStatus(
			r.installationID, r.repo, r.ghDeploymentID,
			state, r.environmentURL, r.logURL, description,
		)
	}, restate.WithName(fmt.Sprintf("github deployment status: %s", state)), restate.WithMaxRetryDuration(30*time.Second))
}

// createStatusReporter builds the appropriate reporter for a deployment.
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

	envLabel := formatEnvironmentLabel(project.Slug, app.Slug, environment.Slug)
	prefix := formatDomainPrefix(project.Slug, app.Slug)
	envURL := fmt.Sprintf("https://%s-%s-%s.%s", prefix, environment.Slug, workspace.Slug, w.defaultDomain)
	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s", w.dashboardURL, workspace.Slug, project.ID, deployment.ID)

	reporter := newGithubStatusReporter(githubStatusReporterConfig{
		GitHub:           w.github,
		DB:               w.db,
		InstallationID:   repoConn.InstallationID,
		Repo:             repoConn.RepositoryFullName,
		CommitSHA:        deployment.GitCommitSha.String,
		EnvironmentLabel: envLabel,
		EnvironmentURL:   envURL,
		LogURL:           logURL,
		DeploymentID:     deployment.ID,
		IsProduction:     environment.Slug == "production",
	})
	reporter.Create(ctx)
	return reporter
}

// formatEnvironmentLabel builds a human-readable label like "project - env"
// or "project/app - env" for non-default apps.
func formatEnvironmentLabel(projectSlug, appSlug, envSlug string) string {
	if appSlug != "default" {
		return projectSlug + "/" + appSlug + " - " + envSlug
	}
	return projectSlug + " - " + envSlug
}

// formatDomainPrefix builds the domain prefix like "project" or "project-app"
// for non-default apps.
func formatDomainPrefix(projectSlug, appSlug string) string {
	if appSlug != "default" {
		return projectSlug + "-" + appSlug
	}
	return projectSlug
}
