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

// deploymentStatusReporter reports deployment progress to an external system.
// Use githubStatusReporter for GitHub-connected deployments, or noopStatusReporter
// for deployments without a GitHub connection.
type deploymentStatusReporter interface {
	Create(ctx restate.ObjectSharedContext)
	Report(ctx restate.ObjectSharedContext, state string, description string)
}

// noopStatusReporter is a no-op implementation for deployments without a GitHub
// repo connection.
type noopStatusReporter struct{}

// NewNoopStatusReporter creates a no-op status reporter for deployments without
// a GitHub repo connection.
func NewNoopStatusReporter() deploymentStatusReporter {
	return noopStatusReporter{}
}

func (noopStatusReporter) Create(_ restate.ObjectSharedContext)                     {}
func (noopStatusReporter) Report(_ restate.ObjectSharedContext, _ string, _ string) {}

// githubStatusReporter wraps GitHub Deployment status reporting with
// fire-and-forget semantics. All GitHub API calls log errors but never
// propagate them — GitHub being down must not block deployments.
type githubStatusReporter struct {
	github             githubclient.GitHubClient
	db                 db.Database
	installationID     int64
	repo               string
	commitSHA          string
	environmentLabel   string
	environmentURL     string
	logURL             string
	deploymentID       string // our internal deployment ID
	githubDeploymentID int64  // set after Create
	isProduction       bool
}

func newGithubStatusReporter(
	github githubclient.GitHubClient,
	database db.Database,
	installationID int64,
	repo string,
	commitSHA string,
	environmentLabel string,
	environmentURL string,
	logURL string,
	deploymentID string,
	isProduction bool,
) *githubStatusReporter {
	return &githubStatusReporter{
		github:           github,
		db:               database,
		installationID:   installationID,
		repo:             repo,
		commitSHA:        commitSHA,
		environmentLabel: environmentLabel,
		environmentURL:   environmentURL,
		logURL:           logURL,
		deploymentID:     deploymentID,
		isProduction:     isProduction,
	}
}

// Create creates the GitHub Deployment and sets the initial status to pending.
func (r *githubStatusReporter) Create(ctx restate.ObjectSharedContext) {
	if r.installationID == 0 || r.repo == "" || r.commitSHA == "" {
		return
	}

	ghDeploymentID, err := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return r.github.CreateDeployment(
			r.installationID,
			r.repo,
			r.commitSHA,
			r.environmentLabel,
			"Deploying...",
			r.isProduction,
		)
	}, restate.WithName("create github deployment"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Error("failed to create GitHub deployment", "error", err, "deployment_id", r.deploymentID)
		return
	}

	r.githubDeploymentID = ghDeploymentID

	_ = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentGithubDeploymentId(runCtx, r.db.RW(), db.UpdateDeploymentGithubDeploymentIdParams{
			GithubDeploymentID: sql.NullInt64{Valid: true, Int64: ghDeploymentID},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:                 r.deploymentID,
		})
	}, restate.WithName("persist github deployment id"), restate.WithMaxRetryDuration(30*time.Second))

	r.Report(ctx, "pending", "Deployment queued")
}

// Report updates the GitHub Deployment status. No-op if Create was not called
// or failed.
func (r *githubStatusReporter) Report(ctx restate.ObjectSharedContext, state string, description string) {
	if r.githubDeploymentID == 0 {
		return
	}

	_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return r.github.CreateDeploymentStatus(
			r.installationID,
			r.repo,
			r.githubDeploymentID,
			state,
			r.environmentURL,
			r.logURL,
			description,
		)
	}, restate.WithName("github deployment status: "+state), restate.WithMaxRetryDuration(30*time.Second))
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
