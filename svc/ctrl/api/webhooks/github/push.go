package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

type pushPayload struct {
	Ref          string           `json:"ref"`
	After        string           `json:"after"`
	Created      bool             `json:"created"`
	Deleted      bool             `json:"deleted"`
	Installation pushInstallation `json:"installation"`
	Repository   pushRepository   `json:"repository"`
	Commits      []pushCommit     `json:"commits"`
	HeadCommit   *pushCommit      `json:"head_commit"`
	Sender       pushSender       `json:"sender"`
}

// pushInstallation, pushRepository, and pushSender are the shared GitHub
// payload primitives; the pull_request payload reuses them.
type pushInstallation struct {
	ID int64 `json:"id"`
}

type pushRepository struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	Fork     bool   `json:"fork"`
}

type pushCommit struct {
	ID        string           `json:"id"`
	Message   string           `json:"message"`
	Timestamp string           `json:"timestamp"`
	Author    pushCommitAuthor `json:"author"`
	Added     []string         `json:"added"`
	Removed   []string         `json:"removed"`
	Modified  []string         `json:"modified"`
}

type pushCommitAuthor struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type pushSender struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

// push parses the push payload, extracts commit metadata, and sends a
// HandlePush request to the GitHubWebhookService in Restate.
func (h *handler) push(ctx context.Context, event webhook.Event, payload pushPayload) error {
	// Deleted branches have no code to build. `created: true` is NOT skipped:
	// GitHub sets it on the first push of a new branch, which is the main way
	// preview deployments get triggered.
	if payload.Deleted {
		return fmt.Errorf("%w: branch delete push for %s", webhook.ErrIgnore, payload.Ref)
	}

	branch := extractBranchFromRef(payload.Ref)
	if branch == "" {
		return fmt.Errorf("%w: non-branch push for %s", webhook.ErrIgnore, payload.Ref)
	}

	gitCommit := extractGitCommitInfo(&payload)

	// Key by installation_id:repo_id for per-repository serialization. Colon
	// delimiter because Restate interprets slashes as path separators.
	objectKey := fmt.Sprintf("%d:%d", payload.Installation.ID, payload.Repository.ID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(h.restate, objectKey)

	deliveryID := event.ID
	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

	changedFiles := collectChangedFiles(payload.Commits)

	_, err := client.HandlePush().Send(ctx, &hydrav1.HandlePushRequest{
		InstallationId:        payload.Installation.ID,
		RepositoryId:          payload.Repository.ID,
		RepositoryFullName:    payload.Repository.FullName,
		Branch:                branch,
		After:                 payload.After,
		CommitMessage:         gitCommit.Message,
		CommitAuthorHandle:    gitCommit.AuthorHandle,
		CommitAuthorAvatarUrl: gitCommit.AuthorAvatarURL,
		CommitTimestamp:       gitCommit.Timestamp.UnixMilli(),
		DeliveryId:            deliveryID,
		ChangedFiles:          changedFiles,
		SenderLogin:           payload.Sender.Login,
	}, sendOpts...)
	if err != nil {
		return fmt.Errorf("enqueue push for %s: %w", payload.Repository.FullName, err)
	}

	logger.Info("GitHub push webhook enqueued to Restate",
		"delivery_id", deliveryID,
		"repository", payload.Repository.FullName,
		"branch", branch,
		"commit_sha", payload.After,
	)
	return nil
}

// extractBranchFromRef returns the branch name from a Git ref, or empty for
// non-branch refs (e.g. tags).
func extractBranchFromRef(ref string) string {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

// extractGitCommitInfo pulls commit metadata from the push payload, preferring
// HeadCommit and falling back to the first commit.
func extractGitCommitInfo(payload *pushPayload) githubclient.CommitInfo {
	headCommit := payload.HeadCommit
	if headCommit == nil && len(payload.Commits) > 0 {
		c := payload.Commits[0]
		headCommit = &pushCommit{
			ID:        c.ID,
			Message:   c.Message,
			Timestamp: c.Timestamp,
			Author:    c.Author,
			Added:     c.Added,
			Removed:   c.Removed,
			Modified:  c.Modified,
		}
	}

	if headCommit == nil {
		return githubclient.CommitInfoFromRaw("", "", "", "", "")
	}

	authorHandle := payload.Sender.Login
	authorAvatar := payload.Sender.AvatarURL
	if authorAvatar == "" {
		authorAvatar = fmt.Sprintf("https://github.com/%s.png", url.PathEscape(authorHandle))
	}

	return githubclient.CommitInfoFromRaw(
		headCommit.ID,
		headCommit.Message,
		authorHandle,
		authorAvatar,
		headCommit.Timestamp,
	)
}

// collectChangedFiles deduplicates file paths across all commits in a push.
func collectChangedFiles(commits []pushCommit) []string {
	seen := make(map[string]struct{})
	for _, c := range commits {
		for _, f := range c.Added {
			seen[f] = struct{}{}
		}
		for _, f := range c.Removed {
			seen[f] = struct{}{}
		}
		for _, f := range c.Modified {
			seen[f] = struct{}{}
		}
	}
	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	return files
}
