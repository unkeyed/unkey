package gitlab

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

// zeroSHA is how GitLab marks a missing side of a push: after == zeroSHA is a
// branch deletion, before == zeroSHA is a branch creation.
const zeroSHA = "0000000000000000000000000000000000000000"

type pushPayload struct {
	Ref          string       `json:"ref"`
	After        string       `json:"after"`
	CheckoutSHA  string       `json:"checkout_sha"`
	UserUsername string       `json:"user_username"`
	UserAvatar   string       `json:"user_avatar"`
	Project      pushProject  `json:"project"`
	Commits      []pushCommit `json:"commits"`
}

type pushProject struct {
	// ID is GitLab's numeric project id. It identifies both the credential
	// scope and the repository, so it fills installation_id and repository_id
	// on the connection row.
	ID                int64  `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}

type pushCommit struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Modified  []string `json:"modified"`
}

// push parses the Push Hook payload and sends a HandlePush request with
// provider="gitlab" to the shared webhook Restate object.
func (h *handler) push(ctx context.Context, event webhook.Event, payload pushPayload) error {
	if payload.After == zeroSHA {
		return fmt.Errorf("%w: branch delete push for %s", webhook.ErrIgnore, payload.Ref)
	}

	branch := extractBranchFromRef(payload.Ref)
	if branch == "" {
		return fmt.Errorf("%w: non-branch push for %s", webhook.ErrIgnore, payload.Ref)
	}

	gitCommit := extractGitCommitInfo(&payload)

	// Key by provider:project_id for per-repository serialization, mirroring
	// GitHub's installation:repo key. The provider prefix keeps GitLab project
	// ids from colliding with GitHub installation ids in the shared object.
	objectKey := fmt.Sprintf("gitlab:%d", payload.Project.ID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(h.restate, objectKey)

	deliveryID := event.ID
	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

	changedFiles := collectChangedFiles(payload.Commits)

	_, err := client.HandlePush().Send(ctx, &hydrav1.HandlePushRequest{
		Provider:              "gitlab",
		InstallationId:        payload.Project.ID,
		RepositoryId:          payload.Project.ID,
		RepositoryFullName:    payload.Project.PathWithNamespace,
		Branch:                branch,
		After:                 payload.After,
		CommitMessage:         gitCommit.Message,
		CommitAuthorHandle:    gitCommit.AuthorHandle,
		CommitAuthorAvatarUrl: gitCommit.AuthorAvatarURL,
		CommitTimestamp:       gitCommit.Timestamp.UnixMilli(),
		DeliveryId:            deliveryID,
		ChangedFiles:          changedFiles,
		SenderLogin:           payload.UserUsername,
	}, sendOpts...)
	if err != nil {
		return fmt.Errorf("enqueue push for %s: %w", payload.Project.PathWithNamespace, err)
	}

	logger.Info("GitLab push webhook enqueued to Restate",
		"delivery_id", deliveryID,
		"repository", payload.Project.PathWithNamespace,
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

// extractGitCommitInfo pulls commit metadata from the push payload. GitLab has
// no head_commit field; the head is the commit whose id matches `after`, with
// the last commit as fallback (GitLab caps the commits array at 20, newest
// last).
func extractGitCommitInfo(payload *pushPayload) githubclient.CommitInfo {
	var head *pushCommit
	for i := range payload.Commits {
		if payload.Commits[i].ID == payload.After {
			head = &payload.Commits[i]
			break
		}
	}
	if head == nil && len(payload.Commits) > 0 {
		head = &payload.Commits[len(payload.Commits)-1]
	}

	if head == nil {
		return githubclient.CommitInfoFromRaw("", "", "", "", "")
	}

	return githubclient.CommitInfoFromRaw(
		head.ID,
		head.Message,
		payload.UserUsername,
		payload.UserAvatar,
		head.Timestamp,
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
