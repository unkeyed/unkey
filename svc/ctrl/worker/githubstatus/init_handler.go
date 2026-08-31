package githubstatus

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Restate K/V state keys.
const (
	stateConfig         = "config"
	stateGHDeploymentID = "github_deployment_id"
	statePRNumber       = "pr_number"
)

// Init creates the GitHub deployment and queues the first PR comment update.
// It is called once per deployment, after the build step.
func (s *Service) Init(ctx restate.ObjectContext, req *hydrav1.GitHubStatusInitRequest) (*hydrav1.GitHubStatusInitResponse, error) {
	if req.GetInstallationId() == 0 || req.GetRepo() == "" {
		return &hydrav1.GitHubStatusInitResponse{}, nil
	}

	// Persist init config so ReportStatus can read it later.
	restate.Set(ctx, stateConfig, req)

	deploymentID := restate.Key(ctx)

	// --- GitHub Deployment ---
	ghDeploymentID := req.GetExistingGithubDeploymentId()
	if ghDeploymentID == 0 && req.GetCommitSha() != "" {
		var err error
		ghDeploymentID, err = restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
			return s.github.CreateDeployment(
				req.GetInstallationId(), req.GetRepo(), req.GetCommitSha(),
				req.GetEnvironmentLabel(), "Deploying...", req.GetIsProduction(),
			)
		}, restate.WithName("create github deployment"), restate.WithMaxRetryDuration(30*time.Second))
		if err != nil {
			logger.Error("failed to create GitHub deployment", "error", err, "deployment_id", deploymentID)
		}
	}

	if ghDeploymentID > 0 {
		restate.Set(ctx, stateGHDeploymentID, ghDeploymentID)

		// Persist to DB so other systems can read it.
		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.db.UpdateDeploymentGithubDeploymentId(runCtx, db.UpdateDeploymentGithubDeploymentIdParams{
				GithubDeploymentID: sql.NullInt64{Valid: true, Int64: ghDeploymentID},
				UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				ID:                 deploymentID,
			})
		}, restate.WithName("persist github deployment id"), restate.WithMaxRetryDuration(30*time.Second)); err != nil {
			logger.Error("failed to persist GitHub deployment ID", "error", err, "deployment_id", deploymentID)
		}
	}

	// --- PR Number ---
	prNumber := int(req.GetPrNumber())
	if prNumber == 0 && req.GetBranch() != "" {
		var err error
		prNumber, err = restate.Run(ctx, func(_ restate.RunContext) (int, error) {
			return s.github.FindPullRequestForBranch(req.GetInstallationId(), req.GetRepo(), req.GetBranch())
		}, restate.WithName("find PR for branch"), restate.WithMaxRetryDuration(5*time.Second))
		if err != nil || prNumber == 0 {
			if err != nil {
				logger.Error("failed to find PR for branch", "error", err, "branch", req.GetBranch())
			}
		}
	}
	if prNumber > 0 {
		restate.Set(ctx, statePRNumber, prNumber)
		sendPullRequestCommentUpdate(ctx, req, int32(prNumber), "Queued")
	}

	// Report initial pending status via the GitHub Deployments API.
	if ghDeploymentID > 0 {
		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return s.github.CreateDeploymentStatus(
				req.GetInstallationId(), req.GetRepo(), ghDeploymentID,
				"pending", req.GetEnvironmentUrl(), req.GetLogUrl(), "Deployment queued",
			)
		}, restate.WithName("github deployment status: pending"), restate.WithMaxRetryDuration(30*time.Second))
	}

	return &hydrav1.GitHubStatusInitResponse{}, nil
}
