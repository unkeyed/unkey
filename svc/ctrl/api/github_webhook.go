package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const maxWebhookBodySize = 2 * 1024 * 1024 // 2 MB

// GitHubWebhook handles incoming GitHub App webhook events by validating
// signatures and dispatching to the GitHubWebhookService in Restate.
// The HTTP handler performs no DB access — all processing happens durably
// inside the Restate virtual object.
type GitHubWebhook struct {
	restate       *restateingress.Client
	webhookSecret string
}

// ServeHTTP validates the webhook signature and dispatches to event-specific
// handlers. Currently supports push events for triggering deployments.
// Unknown event types are acknowledged with 200 OK but not processed.
func (s *GitHubWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Info("GitHub webhook request received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	if r.Method != http.MethodPost {
		logger.Warn("GitHub webhook rejected: method not allowed", "method", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		http.Error(w, "missing X-GitHub-Event header", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		logger.Warn("GitHub webhook rejected: missing signature header")
		http.Error(w, "missing X-Hub-Signature-256 header", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodySize))

	if err != nil {
		logger.Warn("GitHub webhook rejected: failed to read body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if !githubclient.VerifyWebhookSignature(body, signature, s.webhookSecret) {
		logger.Warn("GitHub webhook rejected: invalid signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	logger.Info("GitHub webhook signature verified", "event", event)

	deliveryID := r.Header.Get("X-GitHub-Delivery")

	switch event {
	case "push":
		s.handlePush(r.Context(), w, body, deliveryID)
	case "installation":
		logger.Info("Installation event received")
		w.WriteHeader(http.StatusOK)
	default:
		logger.Info("Unhandled event type", "event", event)
		w.WriteHeader(http.StatusOK)
	}

}

// handlePush parses the push payload, extracts commit metadata, and sends
// a HandlePush request to the GitHubWebhookService in Restate. No DB access
// happens here — the Restate service handles all processing durably.
func (s *GitHubWebhook) handlePush(ctx context.Context, w http.ResponseWriter, body []byte, deliveryID string) {
	var payload pushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.Error("failed to parse push payload", "error", err)
		http.Error(w, "failed to parse push payload", http.StatusBadRequest)
		return
	}

	if payload.Repository.Fork {
		logger.Info("Ignoring push from forked repository", "repository", payload.Repository.FullName)
		w.WriteHeader(http.StatusOK)
		return
	}

	branch := extractBranchFromRef(payload.Ref)
	if branch == "" {
		logger.Info("Ignoring non-branch push", "ref", payload.Ref)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract commit metadata from the payload
	gitCommit := extractGitCommitInfo(&payload)

	// Key by installation_id:repo_id for per-repository serialization.
	// Colon delimiter because Restate interprets slashes as path separators.
	objectKey := fmt.Sprintf("%d:%d", payload.Installation.ID, payload.Repository.ID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(s.restate, objectKey)

	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

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
	}, sendOpts...)
	if err != nil {
		logger.Error("failed to send HandlePush to Restate",
			"error", err,
			"delivery_id", deliveryID,
			"repository", payload.Repository.FullName,
		)
		http.Error(w, "failed to enqueue webhook processing", http.StatusInternalServerError)
		return
	}

	logger.Info("GitHub push webhook enqueued to Restate",
		"delivery_id", deliveryID,
		"repository", payload.Repository.FullName,
		"branch", branch,
		"commit_sha", payload.After,
	)

	w.WriteHeader(http.StatusOK)
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
		}
	}

	if headCommit == nil {
		return githubclient.CommitInfoFromRaw("", "", "", "", "")
	}

	authorHandle := headCommit.Author.Username
	if authorHandle == "" {
		authorHandle = headCommit.Author.Name
	}

	return githubclient.CommitInfoFromRaw(
		headCommit.ID,
		headCommit.Message,
		authorHandle,
		payload.Sender.AvatarURL,
		headCommit.Timestamp,
	)
}
