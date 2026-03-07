package githubwebhook

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

// HandlePush processes a GitHub push event durably via Restate. It looks up
// repo connections with full deploy context (project, environment, app, settings)
// in a single query, creates deployment records, and fires off DeployService.Deploy().
func (s *Service) HandlePush(ctx restate.ObjectContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	logger.Info("handling GitHub push in Restate",
		"delivery_id", req.GetDeliveryId(),
		"repository", req.GetRepositoryFullName(),
		"branch", req.GetBranch(),
		"commit_sha", req.GetAfter(),
		"sender_login", req.GetSenderLogin(),
	)

	branch := req.GetBranch()

	// Single query: connections + apps + projects + environments + build/runtime settings
	// Filters by environment slug based on branch vs project default_branch in SQL.
	contexts, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListRepoConnectionDeployContextsRow, error) {
		return db.Query.ListRepoConnectionDeployContexts(runCtx, s.db.RO(), db.ListRepoConnectionDeployContextsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
			Branch:         branch,
		})
	}, restate.WithName("list deploy contexts"))
	if err != nil {
		return nil, err
	}

	if len(contexts) == 0 {
		logger.Info("no deploy contexts found",
			"installation_id", req.GetInstallationId(),
			"repository_id", req.GetRepositoryId(),
			"branch", req.GetBranch(),
		)
		return &hydrav1.HandlePushResponse{}, nil
	}

	// Single query: all env vars for the matched apps
	allEnvVars, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListEnvVarsForRepoConnectionsRow, error) {
		return db.Query.ListEnvVarsForRepoConnections(runCtx, s.db.RO(), db.ListEnvVarsForRepoConnectionsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
			Branch:         branch,
		})
	}, restate.WithName("list env vars"))
	if err != nil {
		return nil, err
	}

	// Group env vars by app ID
	envVarsByApp := make(map[string][]db.ListEnvVarsForRepoConnectionsRow)
	for _, ev := range allEnvVars {
		envVarsByApp[ev.AppID] = append(envVarsByApp[ev.AppID], ev)
	}

	for _, row := range contexts {
		project := row.Project
		env := row.Environment
		app := row.App
		runtimeSettings := row.AppRuntimeSetting
		buildSettings := row.AppBuildSetting
		repo := row.GithubRepoConnection

		// Build secrets blob from env vars
		appEnvVars := envVarsByApp[app.ID]
		secretsBlob := []byte{}
		if len(appEnvVars) > 0 {
			secretsConfig := &ctrlv1.SecretsConfig{
				Secrets: make(map[string]string, len(appEnvVars)),
			}
			for _, ev := range appEnvVars {
				secretsConfig.Secrets[ev.Key] = ev.Value
			}
			var marshalErr error
			secretsBlob, marshalErr = protojson.Marshal(secretsConfig)
			if marshalErr != nil {
				logger.Error("failed to marshal secrets config", "appId", app.ID, "error", marshalErr)
				continue
			}
		}

		// --- Deployment Protection Check ---
		needsApproval := false
		if project.DeploymentProtection {
			needsApproval = s.requiresApproval(ctx, req, repo)
		}

		// Create deployment record
		deploymentID := uid.New(uid.DeploymentPrefix)
		now := time.Now().UnixMilli()

		commitMessage := req.GetCommitMessage()
		authorHandle := req.GetCommitAuthorHandle()
		authorAvatarURL := req.GetCommitAuthorAvatarUrl()
		commitTimestamp := req.GetCommitTimestamp()

		deploymentStatus := db.DeploymentsStatusPending
		if needsApproval {
			deploymentStatus = db.DeploymentsStatusAwaitingApproval
		}

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Tx(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
				err = db.Query.InsertDeployment(txCtx, tx, db.InsertDeploymentParams{
					ID:                            deploymentID,
					K8sName:                       uid.DNS1035(12),
					WorkspaceID:                   project.WorkspaceID,
					ProjectID:                     project.ID,
					AppID:                         app.ID,
					EnvironmentID:                 env.ID,
					SentinelConfig:                runtimeSettings.SentinelConfig,
					EncryptedEnvironmentVariables: secretsBlob,
					Command:                       runtimeSettings.Command,
					Status:                        deploymentStatus,
					CreatedAt:                     now,
					UpdatedAt:                     sql.NullInt64{Valid: false},
					GitCommitSha:                  sql.NullString{String: req.GetAfter(), Valid: req.GetAfter() != ""},
					GitBranch:                     sql.NullString{String: req.GetBranch(), Valid: req.GetBranch() != ""},
					GitCommitMessage:              sql.NullString{String: commitMessage, Valid: commitMessage != ""},
					GitCommitAuthorHandle:         sql.NullString{String: authorHandle, Valid: authorHandle != ""},
					GitCommitAuthorAvatarUrl:      sql.NullString{String: authorAvatarURL, Valid: authorAvatarURL != ""},
					GitCommitTimestamp:            sql.NullInt64{Int64: commitTimestamp, Valid: commitTimestamp != 0},
					OpenapiSpec:                   sql.NullString{Valid: false},
					CpuMillicores:                 runtimeSettings.CpuMillicores,
					MemoryMib:                     runtimeSettings.MemoryMib,
					Port:                          runtimeSettings.Port,
					ShutdownSignal:                db.DeploymentsShutdownSignal(runtimeSettings.ShutdownSignal),
					Healthcheck:                   runtimeSettings.Healthcheck,
				})
				if err != nil {
					return err
				}

				err = db.Query.InsertDeploymentStep(txCtx, tx, db.InsertDeploymentStepParams{
					WorkspaceID:   app.WorkspaceID,
					ProjectID:     app.ProjectID,
					AppID:         app.ID,
					EnvironmentID: env.ID,
					DeploymentID:  deploymentID,
					Step:          db.DeploymentStepsStepQueued,
					StartedAt:     uint64(now),
				})
				if err != nil {
					return err
				}
				return nil

			})

		}, restate.WithName("insert deployment"))
		if err != nil {
			logger.Error("failed to insert deployment", "appId", app.ID, "error", err)
			continue
		}

		logger.Info("Created deployment record",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", project.ID,
			"app_id", app.ID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
			"branch", req.GetBranch(),
			"environment", env.Slug,
			"needs_approval", needsApproval,
		)

		if needsApproval {
			// Post PR comment if we can find an open PR for this branch
			s.postApprovalComment(ctx, req, repo, deploymentID)
			continue
		}

		// Start deploy workflow keyed by workspace ID, to run 1 concurrent build per workspace for now during beta
		deployClient := hydrav1.NewDeployServiceClient(ctx, app.WorkspaceID)
		deployClient.Deploy().Send(&hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repo.InstallationID,
					Repository:     req.GetRepositoryFullName(),
					CommitSha:      req.GetAfter(),
					ContextPath:    buildSettings.DockerContext,
					DockerfilePath: buildSettings.Dockerfile,
				},
			},
		})

		logger.Info("Deployment workflow started",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", project.ID,
			"app_id", app.ID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
		)
	}

	return &hydrav1.HandlePushResponse{}, nil
}

// requiresApproval determines whether a push needs manual approval.
// Bot accounts (ending in [bot]) and repo collaborators are auto-approved.
func (s *Service) requiresApproval(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	repo db.GithubRepoConnection,
) bool {
	senderLogin := req.GetSenderLogin()

	// No sender info or bot accounts are trusted
	if senderLogin == "" || strings.HasSuffix(senderLogin, "[bot]") {
		return false
	}

	isCollaborator, err := restate.Run(ctx, func(_ restate.RunContext) (bool, error) {
		return s.github.IsCollaborator(
			repo.InstallationID,
			req.GetRepositoryFullName(),
			senderLogin,
		)
	}, restate.WithName("check collaborator status"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		// If we can't check, default to allowing (fail open for collaborator check)
		logger.Error("failed to check collaborator status, allowing deployment",
			"sender", senderLogin,
			"error", err,
		)
		return false
	}

	return !isCollaborator
}

// postApprovalComment finds an open PR for the branch and posts an authorization
// comment. Fire-and-forget — errors are logged but don't block.
func (s *Service) postApprovalComment(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	repo db.GithubRepoConnection,
	deploymentID string,
) {
	prNumber, err := restate.Run(ctx, func(_ restate.RunContext) (int, error) {
		return s.github.FindPullRequestForBranch(
			repo.InstallationID,
			req.GetRepositoryFullName(),
			req.GetBranch(),
		)
	}, restate.WithName("find PR for branch"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Error("failed to find PR for approval comment", "error", err)
		return
	}

	if prNumber == 0 {
		logger.Info("no open PR found for branch, skipping approval comment",
			"branch", req.GetBranch(),
			"deployment_id", deploymentID,
		)
		return
	}

	comment := fmt.Sprintf(
		"**Deployment Authorization Required**\n\n"+
			"This deployment was triggered by @%s who is not a collaborator on this repository.\n\n"+
			"A project member must authorize this deployment before it will proceed.\n\n"+
			"[Authorize Deployment](https://app.unkey.com/deployments/%s/approve)",
		req.GetSenderLogin(),
		deploymentID,
	)

	_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return s.github.CreateIssueComment(
			repo.InstallationID,
			req.GetRepositoryFullName(),
			prNumber,
			comment,
		)
	}, restate.WithName("post approval comment"), restate.WithMaxRetryDuration(30*time.Second))
}
