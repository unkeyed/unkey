package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
	githubverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/github"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// githubWebhook holds the dependencies of the GitHub event handlers. The
// transport concerns (signature verification, routing, metrics, retry
// semantics) live in pkg/webhook; this type only contains business logic.
// The handlers perform no DB access — all processing happens durably inside
// the Restate virtual object they dispatch to.
type githubWebhook struct {
	restate *restateingress.Client
}

// NewGitHubWebhook builds the /webhooks/github handler.
func NewGitHubWebhook(restateClient *restateingress.Client, webhookSecret string) http.Handler {
	s := &githubWebhook{restate: restateClient}
	return webhook.New("github", githubverifier.New(webhookSecret)).
		On(s.handlePush, "push").
		On(s.handlePullRequest, "pull_request").
		// Branch lifecycle events carry no code to deploy; the first push of a
		// new branch arrives as its own push event (created: true), which is
		// what triggers preview deployments.
		On(ignoreEvent, "create", "delete", "installation").
		// GitHub Apps receive every subscribed event type; anything without a
		// handler is deliberately not deployment-relevant.
		Default(ignoreEvent)
}

func ignoreEvent(_ context.Context, event webhook.Event) error {
	return fmt.Errorf("%w: no deployment action for %s events", webhook.ErrIgnore, event.Type)
}

// handlePush parses the push payload, extracts commit metadata, and sends
// a HandlePush request to the GitHubWebhookService in Restate.
func (s *githubWebhook) handlePush(ctx context.Context, event webhook.Event) error {
	var payload pushPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("parse push payload: %w", err)
	}

	// Deleted branches have no code to build — `after` is all zeros and there
	// is nothing to deploy. `created: true` is NOT skipped: GitHub sets it on
	// every first push of a new branch (e.g. `git push -u origin feature`),
	// which is the main way preview deployments get triggered.
	if payload.Deleted {
		return fmt.Errorf("%w: branch delete push for %s", webhook.ErrIgnore, payload.Ref)
	}

	branch := extractBranchFromRef(payload.Ref)
	if branch == "" {
		return fmt.Errorf("%w: non-branch push for %s", webhook.ErrIgnore, payload.Ref)
	}

	// Extract commit metadata from the payload
	gitCommit := extractGitCommitInfo(&payload)

	// Key by installation_id:repo_id for per-repository serialization.
	// Colon delimiter because Restate interprets slashes as path separators.
	objectKey := fmt.Sprintf("%d:%d", payload.Installation.ID, payload.Repository.ID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(s.restate, objectKey)

	deliveryID := event.ID
	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

	// Collect all unique changed files across all commits
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

// handlePullRequest handles pull_request events from forks. Same-repo PRs are
// skipped because the push event already handles those. Fork PRs are dispatched
// through the same HandlePush RPC with IsForkPr=true.
func (s *githubWebhook) handlePullRequest(ctx context.Context, event webhook.Event) error {
	var payload pullRequestPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("parse pull_request payload: %w", err)
	}

	// Only care about new commits (opened or new pushes to the PR)
	if payload.Action != "opened" && payload.Action != "synchronize" {
		return fmt.Errorf("%w: pull_request action %s adds no commits", webhook.ErrIgnore, payload.Action)
	}

	// Same-repo PRs are already handled by the push event — skip to avoid double-deploy
	if payload.PullRequest.Head.Repo.ID == payload.PullRequest.Base.Repo.ID {
		return fmt.Errorf("%w: same-repo pull request, push event handles this", webhook.ErrIgnore)
	}

	pr := payload.PullRequest
	baseRepo := pr.Base.Repo

	objectKey := fmt.Sprintf("%d:%d", payload.Installation.ID, baseRepo.ID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(s.restate, objectKey)

	deliveryID := event.ID
	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

	authorHandle := payload.Sender.Login
	authorAvatar := payload.Sender.AvatarURL
	if authorAvatar == "" {
		authorAvatar = fmt.Sprintf("https://github.com/%s.png", url.PathEscape(authorHandle))
	}

	_, err := client.HandlePush().Send(ctx, &hydrav1.HandlePushRequest{
		InstallationId:         payload.Installation.ID,
		RepositoryId:           baseRepo.ID,
		RepositoryFullName:     baseRepo.FullName,
		Branch:                 pr.Head.Ref,
		After:                  pr.Head.SHA,
		CommitMessage:          pr.Title,
		CommitAuthorHandle:     authorHandle,
		CommitAuthorAvatarUrl:  authorAvatar,
		CommitTimestamp:        time.Now().UnixMilli(),
		DeliveryId:             deliveryID,
		SenderLogin:            payload.Sender.Login,
		IsForkPr:               true,
		PrNumber:               payload.Number,
		ForkRepositoryFullName: pr.Head.Repo.FullName,
	}, sendOpts...)
	if err != nil {
		return fmt.Errorf("enqueue fork PR for %s: %w", baseRepo.FullName, err)
	}

	logger.Info("GitHub fork PR webhook enqueued to Restate",
		"delivery_id", deliveryID,
		"repository", baseRepo.FullName,
		"branch", pr.Head.Ref,
		"commit_sha", pr.Head.SHA,
		"pr_action", payload.Action,
	)
	return nil
}

// extractBranchFromRef extracts the branch name from a Git ref.
// Returns empty string for non-branch refs (e.g., tags).
func extractBranchFromRef(ref string) string {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

// extractGitCommitInfo extracts commit metadata from the push payload,
// preferring HeadCommit when available and falling back to the first commit.
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

// collectChangedFiles deduplicates file paths from all commits in a push.
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
