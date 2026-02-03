package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const maxWebhookBodySize = 2 * 1024 * 1024 // 2 MB

// GitHubWebhook handles GitHub webhook HTTP requests and triggers Restate workflows.
type GitHubWebhook struct {
	db            db.Database
	logger        logging.Logger
	restate       *restateingress.Client
	webhookSecret string
}

// ServeHTTP processes GitHub webhook events.
func (s *GitHubWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodySize))

	if err != nil {
		s.logger.Warn("GitHub webhook rejected: failed to read body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if !githubclient.VerifyWebhookSignature(body, signature, s.webhookSecret) {
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

func (s *GitHubWebhook) handlePush(ctx context.Context, w http.ResponseWriter, body []byte) {
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

	repoConnection, err := db.Query.FindGithubRepoConnection(ctx, s.db.RO(), db.FindGithubRepoConnectionParams{
		InstallationID: payload.Installation.ID,
		RepositoryID:   payload.Repository.ID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			s.logger.Info("No repo connection found for repository", "repository", payload.Repository.FullName)
			w.WriteHeader(http.StatusOK)
			return
		}
		s.logger.Error("failed to find repo connection", "error", err, "repository", payload.Repository.FullName)
		http.Error(w, "failed to find repo connection", http.StatusInternalServerError)
		return
	}

	project, err := db.Query.FindProjectById(ctx, s.db.RO(), repoConnection.ProjectID)
	if err != nil {
		if db.IsNotFound(err) {
			s.logger.Info("No project found for repo connection", "projectId", repoConnection.ProjectID)
			w.WriteHeader(http.StatusOK)
			return
		}
		s.logger.Error("failed to find project", "error", err, "projectId", repoConnection.ProjectID)
		http.Error(w, "failed to find project", http.StatusInternalServerError)
		return
	}

	defaultBranch := "main"
	if project.DefaultBranch.Valid && project.DefaultBranch.String != "" {
		defaultBranch = project.DefaultBranch.String
	}

	// Determine environment based on branch
	envSlug := "preview"
	if branch == defaultBranch {
		envSlug = "production"
	}

	env, err := db.Query.FindEnvironmentByProjectIdAndSlug(ctx, s.db.RO(), db.FindEnvironmentByProjectIdAndSlugParams{
		WorkspaceID: project.WorkspaceID,
		ProjectID:   project.ID,
		Slug:        envSlug,
	})
	if err != nil {
		s.logger.Error("failed to find environment", "error", err, "projectId", project.ID, "envSlug", envSlug)
		http.Error(w, "failed to find environment", http.StatusInternalServerError)
		return
	}

	// Create deployment record
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()
	gitCommit := s.extractGitCommitInfo(&payload, branch)

	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   project.WorkspaceID,
		ProjectID:                     project.ID,
		EnvironmentID:                 env.ID,
		SentinelConfig:                env.SentinelConfig,
		EncryptedEnvironmentVariables: []byte{},
		Command:                       []byte("[]"),
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false},
		GitCommitSha:                  toNullString(payload.After),
		GitBranch:                     toNullString(branch),
		GitCommitMessage:              toNullString(gitCommit.message),
		GitCommitAuthorHandle:         toNullString(gitCommit.authorHandle),
		GitCommitAuthorAvatarUrl:      toNullString(gitCommit.authorAvatarURL),
		GitCommitTimestamp:            toNullInt64(gitCommit.timestamp),
		OpenapiSpec:                   sql.NullString{Valid: false},
		CpuMillicores:                 256,
		MemoryMib:                     256,
	})
	if err != nil {
		s.logger.Error("failed to insert deployment", "error", err)
		http.Error(w, "failed to create deployment", http.StatusInternalServerError)
		return
	}

	s.logger.Info("Created deployment record",
		"deployment_id", deploymentID,
		"project_id", project.ID,
		"repository", payload.Repository.FullName,
		"commit_sha", payload.After,
		"branch", branch,
		"environment", envSlug,
	)

	// Start deploy workflow with GitSource
	deployClient := hydrav1.NewDeploymentServiceIngressClient(s.restate, deploymentID)
	invocation, err := deployClient.Deploy().Send(ctx, &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		Source: &hydrav1.DeployRequest_Git{
			Git: &hydrav1.GitSource{
				InstallationId: repoConnection.InstallationID,
				Repository:     payload.Repository.FullName,
				CommitSha:      payload.After,
				ContextPath:    ".",          // TODO read from project settings
				DockerfilePath: "Dockerfile", // TODO read from project settings
			},
		},
	})
	if err != nil {
		s.logger.Error("failed to start deployment workflow", "error", err)
		http.Error(w, "failed to start workflow", http.StatusInternalServerError)
		return
	}

	s.logger.Info("Deployment workflow started",
		"invocation_id", invocation.Id,
		"deployment_id", deploymentID,
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

type gitCommitInfo struct {
	message         string
	authorHandle    string
	authorAvatarURL string
	timestamp       int64
}

func (s *GitHubWebhook) extractGitCommitInfo(payload *pushPayload, branch string) gitCommitInfo {
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
		return gitCommitInfo{
			message:         "",
			authorHandle:    "",
			authorAvatarURL: "",
			timestamp:       0,
		}
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

	return gitCommitInfo{
		message:         message,
		authorHandle:    authorHandle,
		authorAvatarURL: payload.Sender.AvatarURL,
		timestamp:       timestamp,
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullInt64(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}
