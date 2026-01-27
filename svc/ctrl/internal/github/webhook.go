package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

const maxWebhookBodySize = 2 * 1024 * 1024 // 2 MB

// Webhook handles GitHub webhook HTTP requests and triggers Restate workflows.
type Webhook struct {
	db            db.Database
	logger        logging.Logger
	restate       *restateingress.Client
	webhookSecret string
}

// WebhookConfig holds the configuration for creating a new GitHub webhook service.
type WebhookConfig struct {
	DB            db.Database
	Logger        logging.Logger
	Restate       *restateingress.Client
	WebhookSecret string
}

// NewWebhook creates a new GitHub webhook service.
func NewWebhook(cfg WebhookConfig) (*Webhook, error) {
	if cfg.WebhookSecret == "" {
		return nil, errors.New("webhook secret is required")
	}
	return &Webhook{
		db:            cfg.DB,
		logger:        cfg.Logger,
		restate:       cfg.Restate,
		webhookSecret: cfg.WebhookSecret,
	}, nil
}

// Handler returns an http.Handler that processes GitHub webhook events.
func (s *Webhook) Handler() http.Handler {
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

		r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodySize)

		body := make([]byte, 0, 64*1024)
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
				break
			}
		}

		if !VerifyWebhookSignature(body, signature, s.webhookSecret) {
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

func (s *Webhook) githubClient(projectID string) hydrav1.GitHubServiceIngressClient {
	return hydrav1.NewGitHubServiceIngressClient(s.restate, projectID)
}

type pushPayload struct {
	Ref          string `json:"ref"`
	After        string `json:"after"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
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

func (s *Webhook) handlePush(ctx context.Context, w http.ResponseWriter, body []byte) {
	var payload pushPayload
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

	installation, err := db.Query.FindGithubInstallationByRepo(ctx, s.db.RO(), db.FindGithubInstallationByRepoParams{
		InstallationID: payload.Installation.ID,
		RepositoryID:   payload.Repository.ID,
	})
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

	defaultBranch := "main"
	if project.DefaultBranch.Valid && project.DefaultBranch.String != "" {
		defaultBranch = project.DefaultBranch.String
	}

	s.logger.Info("Starting deployment workflow for push",
		"repository", payload.Repository.FullName,
		"projectId", project.ID,
		"commitSha", payload.After,
		"branch", branch,
		"default_branch", defaultBranch,
	)

	req := &hydrav1.HandlePushRequest{
		InstallationId:     installation.InstallationID,
		RepositoryFullName: payload.Repository.FullName,
		Ref:                payload.Ref,
		CommitSha:          payload.After,
		ProjectId:          project.ID,
		GitCommit:          s.buildGitCommitInfo(&payload, branch),
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
		"commit_sha", payload.After,
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

func (s *Webhook) buildGitCommitInfo(payload *pushPayload, branch string) *hydrav1.GitCommitInfo {
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
