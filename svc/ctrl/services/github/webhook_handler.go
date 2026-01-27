package github

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/github"
)

// maxWebhookBodySize limits webhook payload size to prevent DoS attacks.
// GitHub webhook payloads are typically small (under 100KB), so 2MB is generous.
const maxWebhookBodySize = 2 * 1024 * 1024 // 2 MB

// PushPayload represents the GitHub push event payload.
type PushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"commits"`
	HeadCommit *struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"head_commit"`
	Sender struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"sender"`
}

// WebhookHandler returns an http.Handler that processes GitHub webhook events.
// All requests MUST have a valid X-Hub-Signature-256 header - unsigned requests are rejected.
func (s *Service) WebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("GitHub webhook request received",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		if r.Method != http.MethodPost {
			s.logger.Warn("GitHub webhook rejected: method not allowed", "method", r.Method)
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
			s.logger.Warn("GitHub webhook rejected: missing signature header")
			http.Error(w, "missing X-Hub-Signature-256 header", http.StatusUnauthorized)
			return
		}

		// Limit body size to prevent DoS attacks
		r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodySize)

		body := make([]byte, 0, 64*1024) // Pre-allocate 64KB, typical payload size
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
			}
			if err != nil {
				if err.Error() == "http: request body too large" {
					s.logger.Warn("GitHub webhook rejected: payload too large")
					http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
					return
				}
				break // EOF or other error
			}
		}

		// Always verify signature - this is mandatory, not optional
		if !github.VerifyWebhookSignature(body, signature, s.webhookSecret) {
			s.logger.Warn("GitHub webhook rejected: invalid signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		s.logger.Info("GitHub webhook signature verified", "event", event)

		switch event {
		case "push":
			s.handlePush(r.Context(), w, body)
		case "installation":
			s.logger.Info("Installation event received")
			w.WriteHeader(http.StatusOK)
		default:
			s.logger.Info("Unhandled event type", "event", event)
			w.WriteHeader(http.StatusOK)
		}
	})
}

func (s *Service) handlePush(ctx context.Context, w http.ResponseWriter, body []byte) {
	var payload PushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.logger.Error("failed to parse push payload", "error", err)
		http.Error(w, "failed to parse push payload", http.StatusBadRequest)
		return
	}

	branch := extractBranchFromRef(payload.Ref)
	if branch == "" {
		s.logger.Info("Ignoring non-branch push", "ref", payload.Ref)
		w.WriteHeader(http.StatusOK)
		return
	}

	installation, err := db.Query.FindGithubInstallationByRepo(ctx, s.db.RO(), payload.Repository.FullName)
	if err != nil {
		if db.IsNotFound(err) {
			s.logger.Info("No installation found for repository", "repository", payload.Repository.FullName)
			w.WriteHeader(http.StatusOK)
			return
		}
		s.logger.Error("failed to find installation", "error", err, "repository", payload.Repository.FullName)
		http.Error(w, "failed to find installation", http.StatusInternalServerError)
		return
	}

	project, err := db.Query.FindProjectById(ctx, s.db.RO(), installation.ProjectID)
	if err != nil {
		if db.IsNotFound(err) {
			s.logger.Info("No project found for installation", "projectId", installation.ProjectID)
			w.WriteHeader(http.StatusOK)
			return
		}
		s.logger.Error("failed to find project", "error", err, "projectId", installation.ProjectID)
		http.Error(w, "failed to find project", http.StatusInternalServerError)
		return
	}

	commitSHA := payload.After

	defaultBranch := "main"
	if project.DefaultBranch.Valid && project.DefaultBranch.String != "" {
		defaultBranch = project.DefaultBranch.String
	}

	s.logger.Info("Starting deployment workflow for push",
		"repository", payload.Repository.FullName,
		"projectId", project.ID,
		"commitSha", commitSHA,
		"branch", branch,
		"default_branch", defaultBranch,
	)

	gitCommit := buildGitCommitInfo(&payload, branch)

	req := &hydrav1.HandlePushRequest{
		InstallationId:     installation.InstallationID,
		RepositoryFullName: payload.Repository.FullName,
		Ref:                payload.Ref,
		CommitSha:          commitSHA,
		ProjectId:          project.ID,
		GitCommit:          gitCommit,
		DefaultBranch:      defaultBranch,
	}

	invocation, err := s.githubClient(project.ID).HandlePush().Send(ctx, req)
	if err != nil {
		s.logger.Error("failed to start GitHub push workflow", "error", err)
		http.Error(w, "failed to start workflow", http.StatusInternalServerError)
		return
	}

	s.logger.Info("GitHub push workflow started",
		"invocation_id", invocation.Id,
		"project_id", project.ID,
		"repository", payload.Repository.FullName,
		"commit_sha", commitSHA,
	)

	w.WriteHeader(http.StatusOK)
}

func extractBranchFromRef(ref string) string {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

func buildGitCommitInfo(payload *PushPayload, branch string) *hydrav1.GitCommitInfo {
	headCommit := payload.HeadCommit
	if headCommit == nil && len(payload.Commits) > 0 {
		c := payload.Commits[0]
		headCommit = &struct {
			ID        string `json:"id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			Author    struct {
				Name     string `json:"name"`
				Username string `json:"username"`
			} `json:"author"`
		}{
			ID:        c.ID,
			Message:   c.Message,
			Timestamp: c.Timestamp,
			Author:    c.Author,
		}
	}

	if headCommit == nil {
		return nil
	}

	authorHandle := headCommit.Author.Username
	if authorHandle == "" {
		authorHandle = headCommit.Author.Name
	}

	var timestamp int64
	if t, err := time.Parse(time.RFC3339, headCommit.Timestamp); err == nil {
		timestamp = t.UnixMilli()
	}

	message := headCommit.Message
	if idx := strings.Index(message, "\n"); idx != -1 {
		message = message[:idx]
	}

	return &hydrav1.GitCommitInfo{
		CommitSha:       payload.After,
		CommitMessage:   message,
		AuthorHandle:    authorHandle,
		AuthorAvatarUrl: payload.Sender.AvatarURL,
		Timestamp:       timestamp,
		Branch:          branch,
	}
}
