package githubstatus

import (
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// authorizationStatusContext is the status check name GitHub groups these
// updates under. Both callers write it: Create posts the block, Authorize
// replaces it. Changing the string on one side without the other would leave a
// failing check on the PR forever, so it lives here rather than on the wire.
const authorizationStatusContext = "Unkey Deploy Authorization"

// SetCommitStatus posts a commit status on the deployment's commit.
//
// Best-effort by design: a GitHub outage must not hold up a deployment that is
// otherwise ready to run, so a failure is logged and the handler still
// succeeds. The retry bound keeps a caller's Send from queueing behind a long
// GitHub backoff on this object's key.
func (s *Service) SetCommitStatus(
	ctx restate.ObjectContext,
	req *hydrav1.GitHubStatusCommitStatusRequest,
) (*hydrav1.GitHubStatusCommitStatusResponse, error) {
	deploymentID := restate.Key(ctx)

	// Local development points at real repositories with no GitHub App
	// installed, so writing a status would fail on every push.
	if s.allowUnauthenticatedDeployments {
		logger.Info("skipping commit status: unauthenticated deployments are enabled",
			"deployment_id", deploymentID,
		)
		return &hydrav1.GitHubStatusCommitStatusResponse{}, nil
	}

	state := commitStatusState(req.GetState())
	if req.GetInstallationId() == 0 || req.GetRepo() == "" || req.GetCommitSha() == "" || state == "" {
		logger.Info("skipping commit status: incomplete GitHub context",
			"deployment_id", deploymentID,
			"repo", req.GetRepo(),
			"state", req.GetState().String(),
		)
		return &hydrav1.GitHubStatusCommitStatusResponse{}, nil
	}

	if err := restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return s.github.CreateCommitStatus(
			req.GetInstallationId(),
			req.GetRepo(),
			req.GetCommitSha(),
			state,
			req.GetTargetUrl(),
			req.GetDescription(),
			authorizationStatusContext,
		)
	}, restate.WithName("set commit status: "+state), restate.WithMaxRetryDuration(30*time.Second)); err != nil {
		logger.Error("failed to set commit status",
			"deployment_id", deploymentID,
			"repo", req.GetRepo(),
			"commit_sha", req.GetCommitSha(),
			"state", state,
			"error", err,
		)
	}

	return &hydrav1.GitHubStatusCommitStatusResponse{}, nil
}

// commitStatusState maps the wire enum onto the string the GitHub API expects.
// An unspecified state returns empty, which the caller treats as nothing to
// post rather than guessing.
func commitStatusState(state hydrav1.GitHubCommitStatusState) string {
	switch state {
	case hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_PENDING:
		return "pending"
	case hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_SUCCESS:
		return "success"
	case hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_FAILURE:
		return "failure"
	case hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_ERROR:
		return "error"
	case hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}
