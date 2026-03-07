package deploy

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"

	"github.com/unkeyed/unkey/pkg/logger"
)

// commentMarker is a hidden HTML comment used to identify our bot comment on a PR.
// We search for this to find-and-update instead of creating duplicate comments.
const commentMarker = "<!-- unkey-deploy-comment -->"

// prCommentReporter manages a single Vercel-style deployment comment on a PR.
// It finds an existing comment (by marker) and updates it, or creates a new one.
type prCommentReporter struct {
	github         githubclient.GitHubClient
	installationID int64
	repo           string
	branch         string
	prNumber       int // 0 = no PR found, skip all operations
	commentID      int64
}

// newPRCommentReporter creates a reporter and resolves the PR number for the branch.
func newPRCommentReporter(
	ctx restate.WorkflowSharedContext,
	github githubclient.GitHubClient,
	installationID int64,
	repo string,
	branch string,
) *prCommentReporter {
	r := &prCommentReporter{
		github:         github,
		installationID: installationID,
		repo:           repo,
		branch:         branch,
		prNumber:       0,
		commentID:      0,
	}

	prNumber, err := restate.Run(ctx, func(_ restate.RunContext) (int, error) {
		return github.FindPullRequestForBranch(installationID, repo, branch)
	}, restate.WithName("find PR for deploy comment"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Info("could not find PR for deploy comment", "branch", branch, "error", err)
		return r
	}
	r.prNumber = prNumber

	if prNumber > 0 {
		// Look for existing bot comment
		commentID, findErr := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
			return github.FindBotComment(installationID, repo, prNumber, commentMarker)
		}, restate.WithName("find existing deploy comment"), restate.WithMaxRetryDuration(30*time.Second))
		if findErr != nil {
			logger.Info("could not search for existing deploy comment", "error", findErr)
		} else {
			r.commentID = commentID
		}
	}

	return r
}

// deploymentCommentRow represents one deployment row in the PR comment table.
type deploymentCommentRow struct {
	Environment string
	Status      string
	PreviewURL  string
	UpdatedAt   time.Time
}

// Update posts or updates the PR comment with the current deployment status.
func (r *prCommentReporter) Update(ctx restate.WorkflowSharedContext, rows []deploymentCommentRow) {
	if r.prNumber == 0 {
		return
	}

	body := r.buildComment(rows)

	if r.commentID > 0 {
		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return r.github.UpdateIssueComment(r.installationID, r.repo, r.commentID, body)
		}, restate.WithName("update deploy comment"), restate.WithMaxRetryDuration(30*time.Second))
	} else {
		// Create new comment — we need the ID back for future updates
		type createResult struct {
			ID int64 `json:"id"`
		}
		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return r.github.CreateIssueComment(r.installationID, r.repo, r.prNumber, body)
		}, restate.WithName("create deploy comment"), restate.WithMaxRetryDuration(30*time.Second))
		// We won't get the ID back from CreateIssueComment, so next time we'll find it via marker
	}
}

func (r *prCommentReporter) buildComment(rows []deploymentCommentRow) string {
	comment := commentMarker + "\n"
	comment += "| Environment | Status | Preview | Updated |\n"
	comment += "|:--|:--|:--|:--|\n"

	for _, row := range rows {
		preview := ""
		if row.PreviewURL != "" {
			preview = fmt.Sprintf("[Visit Preview](%s)", row.PreviewURL)
		}
		comment += fmt.Sprintf("| **%s** | %s | %s | %s |\n",
			row.Environment,
			row.Status,
			preview,
			row.UpdatedAt.Format(time.RFC822),
		)
	}

	return comment
}
