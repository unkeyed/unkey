package githubwebhook

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
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
	// Fork PRs always go to preview via the is_fork_pr flag.
	contexts, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListRepoConnectionDeployContextsRow, error) {
		return db.Query.ListRepoConnectionDeployContexts(runCtx, s.db.RO(), db.ListRepoConnectionDeployContextsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
			Branch:         branch,
			IsForkPr:       boolToInt64(req.GetIsForkPr()),
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
			IsForkPr:       boolToInt64(req.GetIsForkPr()),
		})
	}, restate.WithName("list env vars"))
	if err != nil {
		return nil, err
	}

	envVarsByApp := groupEnvVarsByApp(allEnvVars)

	// Webhook payloads don't always include per-commit file lists:
	//   - Fork PRs come through the pull_request webhook which has no commits.
	//   - Created-branch pushes pointing at an already-reachable commit arrive
	//     with an empty commits array.
	// When files aren't available, fetch from the GitHub API so watch-path
	// matching doesn't skip deploys for lack of a diff.
	changedFiles := req.GetChangedFiles()
	if len(changedFiles) == 0 && req.GetAfter() != "" && !s.allowUnauthenticatedDeployments {
		logger.Info("fetching commit files from GitHub",
			"commit_sha", req.GetAfter(),
			"repo", req.GetRepositoryFullName(),
			"installation_id", req.GetInstallationId(),
			"is_fork_pr", req.GetIsForkPr(),
		)
		files, filesErr := restate.Run(ctx, func(_ restate.RunContext) ([]string, error) {
			return s.github.ListCommitFiles(
				req.GetInstallationId(),
				req.GetRepositoryFullName(),
				req.GetAfter(),
			)
		}, restate.WithName("list commit files"))
		if filesErr != nil {
			logger.Error("failed to list commit files, proceeding with empty changed files",
				"commit_sha", req.GetAfter(),
				"error", filesErr,
			)
		} else {
			logger.Info("fetched commit files",
				"commit_sha", req.GetAfter(),
				"changed_files", files,
			)
			changedFiles = files
		}
	}

	for _, row := range contexts {
		project := row.Project
		env := row.Environment
		app := row.App
		repo := row.GithubRepoConnection

		buildSettings := row.AppBuildSetting

		// Watch paths: skip if configured patterns don't match changed files
		if !match.MatchWatchPaths(buildSettings.WatchPaths, changedFiles) {
			logger.Info("skipping deployment: watch paths don't match changed files",
				"app_id", app.ID,
				"watch_paths", buildSettings.WatchPaths,
				"changed_files", changedFiles,
			)

			// Create skipped deployment record for visibility
			_, _ = restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
				return insertDeploymentRecord(runCtx, s.db.RW(), row, req, []byte{}, db.DeploymentsStatusSkipped)
			}, restate.WithName("insert skipped deployment"))
			continue
		}

		secretsBlob, marshalErr := buildSecretsBlob(envVarsByApp[app.ID])
		if marshalErr != nil {
			logger.Error("failed to marshal secrets config", "appId", app.ID, "error", marshalErr)
			continue
		}

		// Approval decision is independent of allowUnauthenticatedDeployments:
		// the flag only controls whether we reach out to GitHub (e.g. to post
		// the "awaiting authorization" commit status — see blockDeploymentForApproval).
		// Fork PRs run external code and must always be gated, even in dev.
		needsApproval := s.requiresApproval(ctx, req, repo)

		status := db.DeploymentsStatusPending
		if needsApproval {
			status = db.DeploymentsStatusAwaitingApproval
		}

		deploymentID, insertErr := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return insertDeploymentRecord(runCtx, s.db.RW(), row, req, secretsBlob, status)
		}, restate.WithName("insert deployment"))
		if insertErr != nil {
			logger.Error("failed to insert deployment", "appId", app.ID, "error", insertErr)
			continue
		}

		logger.Info("created deployment record",
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
			if blockErr := s.blockDeploymentForApproval(ctx, req, project, repo, deploymentID); blockErr != nil {
				return nil, blockErr
			}
			continue
		}

		// Keyed by deployment_id — each deployment is its own isolated workflow.
		// Workspace-wide build concurrency is capped by BuildSlotService.
		deployClient := hydrav1.NewDeployServiceClient(ctx, deploymentID)
		invocation := deployClient.Deploy().Send(&hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repo.InstallationID,
					Repository:     repo.RepositoryFullName,
					CommitSha:      req.GetAfter(),
					ContextPath:    row.AppBuildSetting.DockerContext,
					DockerfilePath: row.AppBuildSetting.Dockerfile,
					PrNumber:       req.GetPrNumber(),
				},
			},
		})

		// Persist the invocation ID so the deployment can be cancelled later.
		// Restate always returns a non-empty invocation ID on a successful Send;
		// an empty value indicates a bug in our send path or the SDK.
		invocationID := invocation.GetInvocationId()
		if invocationID == "" {
			return nil, fmt.Errorf("restate returned empty invocation id for deployment %s", deploymentID)
		}
		_ = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.UpdateDeploymentInvocationID(runCtx, s.db.RW(), db.UpdateDeploymentInvocationIDParams{
				ID:           deploymentID,
				InvocationID: sql.NullString{Valid: true, String: invocationID},
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("persist invocation id"))

		logger.Info("deployment workflow started",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", project.ID,
			"app_id", app.ID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
			"invocation_id", invocationID,
		)

		_ = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			if cancelErr := s.dedup.CancelOlderSiblings(runCtx, dedup.Newer{
				ID:            deploymentID,
				AppID:         app.ID,
				EnvironmentID: env.ID,
				GitBranch:     req.GetBranch(),
				CreatedAt:     time.Now().UnixMilli(),
			}); cancelErr != nil {
				logger.Error("failed to cancel superseded siblings",
					"deployment_id", deploymentID,
					"error", cancelErr,
				)
			}
			return nil
		}, restate.WithName("cancel superseded siblings"))
	}

	return &hydrav1.HandlePushResponse{}, nil
}

// requiresApproval determines whether a push needs manual approval.
// Fork PRs always require approval. Non-fork pushes are auto-approved because
// GitHub already enforces write access — if someone can push to the repo, they
// are authorized.
//
// Set FORCE_DEPLOYMENT_APPROVAL=true to require approval for all pushes.
// This is useful for testing the approval flow locally.
func (s *Service) requiresApproval(
	_ restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	_ db.GithubRepoConnection,
) bool {
	if os.Getenv("FORCE_DEPLOYMENT_APPROVAL") == "true" {
		logger.Info("FORCE_DEPLOYMENT_APPROVAL is set, requiring approval",
			"sender", req.GetSenderLogin(),
		)
		return true
	}

	// Fork PRs always require approval — external code must never auto-deploy.
	if req.GetIsForkPr() {
		logger.Info("fork PR deployment requires approval",
			"sender", req.GetSenderLogin(),
		)
		return true
	}

	// Non-fork pushes: GitHub already verified the pusher has write access to
	// the repo, so there is no reason to gate the deployment behind approval.
	return false
}

// insertDeploymentRecord creates a deployment and its initial queued step in a single transaction.
func insertDeploymentRecord(
	ctx context.Context,
	rw *db.Replica,
	row db.ListRepoConnectionDeployContextsRow,
	req *hydrav1.HandlePushRequest,
	secretsBlob []byte,
	status db.DeploymentsStatus,
) (string, error) {
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	project := row.Project
	env := row.Environment
	app := row.App
	runtimeSettings := row.AppRuntimeSetting

	commitSHA := req.GetAfter()
	branch := req.GetBranch()
	commitMessage := req.GetCommitMessage()
	authorHandle := req.GetCommitAuthorHandle()
	authorAvatarURL := req.GetCommitAuthorAvatarUrl()
	commitTimestamp := req.GetCommitTimestamp()

	err := db.Tx(ctx, rw, func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.Query.InsertDeployment(txCtx, tx, db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   project.WorkspaceID,
			ProjectID:                     project.ID,
			AppID:                         app.ID,
			EnvironmentID:                 env.ID,
			SentinelConfig:                runtimeSettings.SentinelConfig,
			EncryptedEnvironmentVariables: secretsBlob,
			Command:                       runtimeSettings.Command,
			Status:                        status,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false},
			GitCommitSha:                  sql.NullString{String: commitSHA, Valid: commitSHA != ""},
			GitBranch:                     sql.NullString{String: branch, Valid: branch != ""},
			GitCommitMessage:              sql.NullString{String: commitMessage, Valid: commitMessage != ""},
			GitCommitAuthorHandle:         sql.NullString{String: authorHandle, Valid: authorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: authorAvatarURL, Valid: authorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: commitTimestamp, Valid: commitTimestamp != 0},
			CpuMillicores:                 runtimeSettings.CpuMillicores,
			MemoryMib:                     runtimeSettings.MemoryMib,
			StorageMib:                    runtimeSettings.StorageMib,
			Port:                          runtimeSettings.Port,
			ShutdownSignal:                db.DeploymentsShutdownSignal(runtimeSettings.ShutdownSignal),
			UpstreamProtocol:              db.DeploymentsUpstreamProtocol(runtimeSettings.UpstreamProtocol),
			Healthcheck:                   runtimeSettings.Healthcheck,
			PrNumber:                      sql.NullInt64{Int64: req.GetPrNumber(), Valid: req.GetPrNumber() != 0},
			ForkRepositoryFullName:        sql.NullString{String: req.GetForkRepositoryFullName(), Valid: req.GetForkRepositoryFullName() != ""},
			DeploymentTrigger:             db.DeploymentsTriggerGithub,
			TriggeredBy:                   sql.NullString{String: req.GetSenderLogin(), Valid: req.GetSenderLogin() != ""},
			TriggerReason:                 sql.NullString{Valid: false},
		}); txErr != nil {
			return txErr
		}

		return db.Query.InsertDeploymentStep(txCtx, tx, db.InsertDeploymentStepParams{
			WorkspaceID:   app.WorkspaceID,
			ProjectID:     app.ProjectID,
			AppID:         app.ID,
			EnvironmentID: env.ID,
			DeploymentID:  deploymentID,
			Step:          db.DeploymentStepsStepQueued,
			StartedAt:     uint64(now),
		})
	})
	if err != nil {
		return "", err
	}
	return deploymentID, nil
}

// buildSecretsBlob marshals environment variables into a protobuf SecretsConfig blob.
func buildSecretsBlob(envVars []db.ListEnvVarsForRepoConnectionsRow) ([]byte, error) {
	if len(envVars) == 0 {
		return []byte{}, nil
	}

	secretsConfig := &ctrlv1.SecretsConfig{
		Secrets: make(map[string]string, len(envVars)),
	}
	for _, ev := range envVars {
		secretsConfig.Secrets[ev.Key] = ev.Value
	}
	return protojson.Marshal(secretsConfig)
}

// groupEnvVarsByApp groups environment variables by app ID for efficient lookup.
func groupEnvVarsByApp(envVars []db.ListEnvVarsForRepoConnectionsRow) map[string][]db.ListEnvVarsForRepoConnectionsRow {
	result := make(map[string][]db.ListEnvVarsForRepoConnectionsRow)
	for _, ev := range envVars {
		result[ev.AppID] = append(result[ev.AppID], ev)
	}
	return result
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
