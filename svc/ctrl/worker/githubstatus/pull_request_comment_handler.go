package githubstatus

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// findResult keeps the comment ID and body in one Restate journal entry.
type findResult struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// UpsertPullRequestComment serializes all updates to one PR's shared deploy
// comment so concurrent deployments cannot overwrite each other's rows.
func (s *Service) UpsertPullRequestComment(ctx restate.ObjectContext, req *hydrav1.GitHubPullRequestCommentRequest) (*hydrav1.GitHubPullRequestCommentResponse, error) {
	config := req.GetConfig()
	if config == nil || config.GetInstallationId() == 0 || config.GetRepo() == "" || req.GetPrNumber() <= 0 {
		return &hydrav1.GitHubPullRequestCommentResponse{}, nil
	}

	current, err := restate.Run(ctx, func(_ restate.RunContext) (findResult, error) {
		id, body, findErr := s.github.FindBotComment(config.GetInstallationId(), config.GetRepo(), int(req.GetPrNumber()), prCommentMainMarker)
		return findResult{ID: id, Body: body}, findErr
	}, restate.WithName("find deploy comment"), restate.WithMaxRetryDuration(5*time.Second))
	if err != nil {
		logger.Error("failed to find PR deploy comment", "error", err, "repo", config.GetRepo(), "pr", req.GetPrNumber())
		return &hydrav1.GitHubPullRequestCommentResponse{}, nil
	}

	rowKey := config.GetCommentRowKey()
	if rowKey == "" {
		rowKey = config.GetAppSlug() + ":" + config.GetEnvSlug()
	}
	row := buildRow(rowKey, config.GetProjectSlug(), config.GetAppSlug(), config.GetEnvSlug(), config.GetEnvironmentUrl(), config.GetLogUrl(), req.GetStatus())

	if current.ID == 0 {
		_, err = restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
			return s.github.CreateIssueComment(config.GetInstallationId(), config.GetRepo(), int(req.GetPrNumber()), buildFullComment(row))
		}, restate.WithName("create deploy comment"), restate.WithMaxRetryDuration(5*time.Second))
	} else {
		err = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return s.github.UpdateIssueComment(
				config.GetInstallationId(),
				config.GetRepo(),
				current.ID,
				upsertRow(rowKey, config.GetAppSlug(), config.GetEnvSlug(), current.Body, row),
			)
		}, restate.WithName("update deploy comment"), restate.WithMaxRetryDuration(5*time.Second))
	}
	if err != nil {
		logger.Error("failed to upsert PR deploy comment", "error", err, "repo", config.GetRepo(), "pr", req.GetPrNumber())
	}

	return &hydrav1.GitHubPullRequestCommentResponse{}, nil
}

// sendPullRequestCommentUpdate sends the update to the PR-keyed object that
// serializes changes from otherwise independent deployment workflows.
func sendPullRequestCommentUpdate(ctx restate.ObjectContext, config *hydrav1.GitHubStatusInitRequest, prNumber int32, status string) {
	key := pullRequestCommentObjectKey(config.GetInstallationId(), config.GetRepositoryId(), config.GetRepo(), prNumber)
	hydrav1.NewGitHubPullRequestCommentServiceClient(ctx, key).UpsertPullRequestComment().Send(&hydrav1.GitHubPullRequestCommentRequest{
		Config:   config,
		PrNumber: prNumber,
		Status:   status,
	})
}

// pullRequestCommentObjectKey uses GitHub's stable repository ID and falls back
// to a case-insensitive name hash for requests started before that ID existed.
func pullRequestCommentObjectKey(installationID, repositoryID int64, repo string, prNumber int32) string {
	repoIdentity := fmt.Sprintf("id:%d", repositoryID)
	if repositoryID == 0 {
		repoHash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(repo))))
		repoIdentity = fmt.Sprintf("name:%x", repoHash)
	}

	return fmt.Sprintf("%d:%s:%d", installationID, repoIdentity, prNumber)
}
