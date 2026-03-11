package deploy

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

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
