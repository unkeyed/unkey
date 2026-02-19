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
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
	"google.golang.org/protobuf/encoding/protojson"
)

const maxWebhookBodySize = 2 * 1024 * 1024 // 2 MB

// GitHubWebhook handles incoming GitHub App webhook events and triggers
// deployment workflows via Restate. It validates webhook signatures using
// the configured secret before processing any events.
type GitHubWebhook struct {
	db            db.Database
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

	switch event {
	case "push":
		s.handlePush(r.Context(), w, body)
	case "installation":
		logger.Info("Installation event received")
		w.WriteHeader(http.StatusOK)
	default:
		logger.Info("Unhandled event type", "event", event)
		w.WriteHeader(http.StatusOK)
	}
}

// handlePush processes push events by creating a deployment record and
// starting the deploy workflow. Maps branches to environments: the project's
// default branch deploys to production, all others to preview.
func (s *GitHubWebhook) handlePush(ctx context.Context, w http.ResponseWriter, body []byte) {
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

	repoConnections, err := db.Query.ListGithubRepoConnections(ctx, s.db.RO(), db.ListGithubRepoConnectionsParams{
		InstallationID: payload.Installation.ID,
		RepositoryID:   payload.Repository.ID,
	})
	if err != nil {
		logger.Error("failed to find repo connections", "error", err, "repository", payload.Repository.FullName)
		http.Error(w, "failed to find repo connections", http.StatusInternalServerError)
		return
	}

	for _, repo := range repoConnections {

		project, err := db.Query.FindProjectById(ctx, s.db.RO(), repo.ProjectID)
		if err != nil {
			if db.IsNotFound(err) {
				logger.Info("No project found for repo connection", "projectId", repo.ProjectID)
				continue
			}
			logger.Error("failed to find project", "error", err, "projectId", repo.ProjectID)
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
			logger.Error("failed to find environment", "error", err, "projectId", project.ID, "envSlug", envSlug)
			http.Error(w, "failed to find environment", http.StatusInternalServerError)
			return
		}

		// Look up default app for the project
		appRow, err := db.Query.FindAppByProjectAndSlug(ctx, s.db.RO(), db.FindAppByProjectAndSlugParams{
			ProjectID: project.ID,
			Slug:      "default",
		})
		if err != nil {
			logger.Error("failed to find default app", "error", err, "projectId", project.ID)
			http.Error(w, "failed to find default app", http.StatusInternalServerError)
			return
		}
		app := appRow.App

		// Fetch app-scoped runtime and build settings
		runtimeRow, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, s.db.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID:         app.ID,
			EnvironmentID: env.ID,
		})
		if err != nil {
			logger.Error("failed to find app runtime settings", "error", err, "appId", app.ID, "envId", env.ID)
			http.Error(w, "failed to find app runtime settings", http.StatusInternalServerError)
			return
		}
		runtimeSettings := runtimeRow.AppRuntimeSetting

		buildRow, err := db.Query.FindAppBuildSettingsByAppAndEnv(ctx, s.db.RO(), db.FindAppBuildSettingsByAppAndEnvParams{
			AppID:         app.ID,
			EnvironmentID: env.ID,
		})
		if err != nil {
			logger.Error("failed to find app build settings", "error", err, "appId", app.ID, "envId", env.ID)
			http.Error(w, "failed to find app build settings", http.StatusInternalServerError)
			return
		}
		buildSettings := buildRow.AppBuildSetting

		// Fetch app-scoped environment variables and build secrets blob
		envVars, err := db.Query.FindAppEnvVarsByAppAndEnv(ctx, s.db.RO(), db.FindAppEnvVarsByAppAndEnvParams{
			AppID:         app.ID,
			EnvironmentID: env.ID,
		})
		if err != nil {
			logger.Error("failed to fetch environment variables", "error", err)
			http.Error(w, "failed to fetch environment variables", http.StatusInternalServerError)
			return
		}

		secretsBlob := []byte{}
		if len(envVars) > 0 {
			secretsConfig := &ctrlv1.SecretsConfig{
				Secrets: make(map[string]string, len(envVars)),
			}
			for _, ev := range envVars {
				secretsConfig.Secrets[ev.Key] = ev.Value
			}

			secretsBlob, err = protojson.Marshal(secretsConfig)
			if err != nil {
				logger.Error("failed to marshal secrets config", "error", err)
				http.Error(w, "failed to marshal secrets", http.StatusInternalServerError)
				return
			}
		}

		// Create deployment record
		deploymentID := uid.New(uid.DeploymentPrefix)
		now := time.Now().UnixMilli()
		gitCommit := s.extractGitCommitInfo(&payload)

		err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   project.WorkspaceID,
			ProjectID:                     project.ID,
			AppID:                         app.ID,
			EnvironmentID:                 env.ID,
			SentinelConfig:                runtimeSettings.SentinelConfig,
			EncryptedEnvironmentVariables: secretsBlob,
			Command:                       runtimeSettings.Command,
			Status:                        db.DeploymentsStatusPending,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false},
			GitCommitSha:                  sql.NullString{String: payload.After, Valid: payload.After != ""},
			GitBranch:                     sql.NullString{String: branch, Valid: branch != ""},
			GitCommitMessage:              sql.NullString{String: gitCommit.Message, Valid: gitCommit.Message != ""},
			GitCommitAuthorHandle:         sql.NullString{String: gitCommit.AuthorHandle, Valid: gitCommit.AuthorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: gitCommit.AuthorAvatarURL, Valid: gitCommit.AuthorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: gitCommit.Timestamp.UnixMilli(), Valid: !gitCommit.Timestamp.IsZero()},
			OpenapiSpec:                   sql.NullString{Valid: false},
			CpuMillicores:                 runtimeSettings.CpuMillicores,
			MemoryMib:                     runtimeSettings.MemoryMib,
			Port:                          runtimeSettings.Port,
			ShutdownSignal:                db.DeploymentsShutdownSignal(runtimeSettings.ShutdownSignal),
			Healthcheck:                   runtimeSettings.Healthcheck,
		})
		if err != nil {
			logger.Error("failed to insert deployment", "error", err)
			http.Error(w, "failed to create deployment", http.StatusInternalServerError)
			return
		}

		logger.Info("Created deployment record",
			"deployment_id", deploymentID,
			"project_id", project.ID,
			"repository", payload.Repository.FullName,
			"commit_sha", payload.After,
			"branch", branch,
			"environment", envSlug,
		)

		// Start deploy workflow with GitSource, keyed by app ID
		deployClient := hydrav1.NewDeployServiceIngressClient(s.restate, app.ID)
		invocation, err := deployClient.Deploy().Send(ctx, &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repo.InstallationID,
					Repository:     payload.Repository.FullName,
					CommitSha:      payload.After,
					ContextPath:    buildSettings.DockerContext,
					DockerfilePath: buildSettings.Dockerfile,
				},
			},
		})
		if err != nil {
			logger.Error("failed to start deployment workflow", "error", err)
			http.Error(w, "failed to start workflow", http.StatusInternalServerError)
			return
		}

		logger.Info("Deployment workflow started",
			"invocation_id", invocation.Id,
			"deployment_id", deploymentID,
			"project_id", project.ID,
			"repository", payload.Repository.FullName,
			"commit_sha", payload.After,
		)
	}

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
func (s *GitHubWebhook) extractGitCommitInfo(payload *pushPayload) githubclient.CommitInfo {
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
