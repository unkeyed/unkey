package gitea

import (
	"context"
	"fmt"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
)

// Gitea push payloads are deliberately GitHub-compatible: refs/heads/ refs,
// per-commit added/removed/modified file lists, head_commit, numeric
// repository ids.
type pushPayload struct {
	Ref        string       `json:"ref"`
	After      string       `json:"after"`
	Repository pushRepo     `json:"repository"`
	Commits    []pushCommit `json:"commits"`
	HeadCommit *pushCommit  `json:"head_commit"`
	Sender     pushSender   `json:"sender"`
}

type pushRepo struct {
	// ID is only unique within one Gitea instance. The POC tolerates the
	// cross-instance collision risk; the proper implementation must include
	// the instance host in the connection identity.
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type pushCommit struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Modified  []string `json:"modified"`
}

type pushSender struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

// push parses the push payload and sends a HandlePush request with
// provider="gitea" to the shared webhook Restate object.
func (h *handler) push(ctx context.Context, event webhook.Event, payload pushPayload) error {
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	if branch == payload.Ref {
		return fmt.Errorf("%w: non-branch push for %s", webhook.ErrIgnore, payload.Ref)
	}
	// Branch deletions push an all-zero after sha; nothing to build.
	if payload.After == strings.Repeat("0", 40) {
		return fmt.Errorf("%w: branch delete push for %s", webhook.ErrIgnore, payload.Ref)
	}

	head := payload.HeadCommit
	if head == nil && len(payload.Commits) > 0 {
		head = &payload.Commits[0]
	}
	commit := githubclient.CommitInfoFromRaw("", "", payload.Sender.Login, payload.Sender.AvatarURL, "")
	if head != nil {
		commit = githubclient.CommitInfoFromRaw(
			head.ID,
			head.Message,
			payload.Sender.Login,
			payload.Sender.AvatarURL,
			head.Timestamp,
		)
	}

	changedFiles := collectChangedFiles(payload.Commits)

	// Key by provider:repo id for per-repository serialization, mirroring the
	// other providers. Repo ids are per-instance, see pushRepo.ID.
	objectKey := fmt.Sprintf("gitea:%d", payload.Repository.ID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(h.restate, objectKey)

	deliveryID := event.ID
	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

	_, err := client.HandlePush().Send(ctx, &hydrav1.HandlePushRequest{
		Provider:              "gitea",
		InstallationId:        payload.Repository.ID,
		RepositoryId:          payload.Repository.ID,
		RepositoryFullName:    payload.Repository.FullName,
		Branch:                branch,
		After:                 payload.After,
		CommitMessage:         commit.Message,
		CommitAuthorHandle:    commit.AuthorHandle,
		CommitAuthorAvatarUrl: commit.AuthorAvatarURL,
		CommitTimestamp:       commit.Timestamp.UnixMilli(),
		DeliveryId:            deliveryID,
		ChangedFiles:          changedFiles,
		SenderLogin:           payload.Sender.Login,
	}, sendOpts...)
	if err != nil {
		return fmt.Errorf("enqueue push for %s: %w", payload.Repository.FullName, err)
	}

	logger.Info("Gitea push webhook enqueued to Restate",
		"delivery_id", deliveryID,
		"repository", payload.Repository.FullName,
		"branch", branch,
		"commit_sha", payload.After,
	)
	return nil
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
