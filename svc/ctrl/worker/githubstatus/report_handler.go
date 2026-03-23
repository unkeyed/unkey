package githubstatus

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// githubDeploymentStateToString maps the proto enum to the GitHub API string.
var githubDeploymentStateToString = map[hydrav1.GitHubDeploymentState]string{
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_PENDING:     "pending",
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_IN_PROGRESS: "in_progress",
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_SUCCESS:     "success",
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_FAILURE:     "failure",
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_ERROR:       "error",
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_INACTIVE:    "inactive",
	hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_QUEUED:      "queued",
}

// ReportStatus updates both the GitHub deployment status and the PR comment.
// All calls are fire-and-forget — errors are logged, never propagated.
func (s *Service) ReportStatus(ctx restate.ObjectContext, req *hydrav1.GitHubStatusReportRequest) (*hydrav1.GitHubStatusReportResponse, error) {
	config, err := restate.Get[*hydrav1.GitHubStatusInitRequest](ctx, stateConfig)
	if err != nil || config == nil {
		return &hydrav1.GitHubStatusReportResponse{}, nil
	}

	ghDeploymentID, _ := restate.Get[int64](ctx, stateGHDeploymentID)
	commentID, _ := restate.Get[int64](ctx, statePRCommentID)
	prNumber, _ := restate.Get[int](ctx, statePRNumber)

	stateStr, ok := githubDeploymentStateToString[req.GetState()]
	if !ok {
		stateStr = "in_progress"
	}

	deploymentID := restate.Key(ctx)

	// --- GitHub Deployment Status ---
	if ghDeploymentID > 0 {
		if err := restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return s.github.CreateDeploymentStatus(
				config.GetInstallationId(), config.GetRepo(), ghDeploymentID,
				stateStr, config.GetEnvironmentUrl(), config.GetLogUrl(), req.GetDescription(),
			)
		}, restate.WithName(fmt.Sprintf("github deployment status: %s", stateStr)), restate.WithMaxRetryDuration(30*time.Second)); err != nil {
			logger.Error("failed to report GitHub deployment status", "error", err, "deployment_id", deploymentID, "state", stateStr)
		}
	}

	// --- PR Comment ---
	if commentID > 0 && prNumber > 0 {
		label := stateLabel(stateStr)
		row := buildRow(config.GetProjectSlug(), config.GetAppSlug(), config.GetEnvSlug(), config.GetEnvironmentUrl(), config.GetLogUrl(), label)

		current, findErr := restate.Run(ctx, func(_ restate.RunContext) (findResult, error) {
			id, body, e := s.github.FindBotComment(config.GetInstallationId(), config.GetRepo(), prNumber, prCommentMainMarker)
			return findResult{ID: id, Body: body}, e
		}, restate.WithName("read deploy comment"), restate.WithMaxRetryDuration(5*time.Second))
		if findErr != nil || current.ID == 0 {
			return &hydrav1.GitHubStatusReportResponse{}, nil
		}

		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return s.github.UpdateIssueComment(config.GetInstallationId(), config.GetRepo(), commentID, upsertRow(config.GetAppSlug(), config.GetEnvSlug(), current.Body, row))
		}, restate.WithName(fmt.Sprintf("update deploy comment: %s", stateStr)), restate.WithMaxRetryDuration(5*time.Second))
	}

	return &hydrav1.GitHubStatusReportResponse{}, nil
}
