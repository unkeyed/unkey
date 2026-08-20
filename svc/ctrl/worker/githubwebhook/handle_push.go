package githubwebhook

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
	// Selects the production environment for the default branch and preview for others.
	// Fork PRs always go to preview via the is_fork_pr flag.
	contexts, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListRepoConnectionDeployContextsRow, error) {
		return s.db.ListRepoConnectionDeployContexts(runCtx, db.ListRepoConnectionDeployContextsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
			Branch:         sql.NullString{String: branch, Valid: branch != ""},
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

	// Gate before loading env vars, calling GitHub, or writing even a skipped
	// deployment row. A policy rejection is a successful no-op so Restate does
	// not retry a permanently ineligible workspace and stall the repository.
	entitlements := make(map[string]db.FindWorkspaceDeployEntitlementRow)
	eligibleContexts := make([]db.ListRepoConnectionDeployContextsRow, 0, len(contexts))
	for _, row := range contexts {
		entitlement, ok := entitlements[row.ProjectWorkspaceID]
		if !ok {
			entitlement, err = restate.Run(ctx, func(runCtx restate.RunContext) (db.FindWorkspaceDeployEntitlementRow, error) {
				loaded, loadErr := s.db.FindWorkspaceDeployEntitlement(runCtx, row.ProjectWorkspaceID)
				if db.IsNotFound(loadErr) {
					return db.FindWorkspaceDeployEntitlementRow{
						Plan:           sql.NullString{},
						PlanOverride:   sql.NullString{},
						SpendSuspended: sql.NullBool{},
					}, nil
				}
				return loaded, loadErr
			}, restate.WithName("load workspace deploy entitlement "+row.ProjectWorkspaceID))
			if err != nil {
				return nil, err
			}
			entitlements[row.ProjectWorkspaceID] = entitlement
		}

		if !deploygate.Entitled(entitlement.Plan, entitlement.PlanOverride) {
			if s.enforceDeployGate {
				logger.Info("skipping deployment: workspace has no Compute plan",
					"event", "deploy_gate.blocked",
					"reason", "no_plan",
					"workspace_id", row.ProjectWorkspaceID,
					"project_id", row.ProjectID,
					"app_id", row.AppID,
					"delivery_id", req.GetDeliveryId(),
				)
				continue
			}
			logger.Warn("deploy gate would block GitHub deployment",
				"event", "deploy_gate.would_block",
				"workspaceId", row.ProjectWorkspaceID,
				"projectId", row.ProjectID,
				"appId", row.AppID,
			)
		}
		if entitlement.SpendSuspended.Bool {
			logger.Info("skipping deployment: workspace is spend suspended",
				"event", "deploy_gate.blocked",
				"reason", "spend_suspended",
				"workspace_id", row.ProjectWorkspaceID,
				"project_id", row.ProjectID,
				"app_id", row.AppID,
				"delivery_id", req.GetDeliveryId(),
			)
			continue
		}

		eligibleContexts = append(eligibleContexts, row)
	}
	contexts = eligibleContexts
	if len(contexts) == 0 {
		return &hydrav1.HandlePushResponse{}, nil
	}

	// Single query: all env vars for the matched apps
	allEnvVars, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListEnvVarsForRepoConnectionsRow, error) {
		return s.db.ListEnvVarsForRepoConnections(runCtx, db.ListEnvVarsForRepoConnectionsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
			Branch:         sql.NullString{String: branch, Valid: branch != ""},
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
		// The dashboard reads this reason off the skipped deployment, so every skip
		// says why rather than leaving the push with no record at all.
		skipDeployment := func(reason string) {
			if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				_, err := insertDeploymentRecord(runCtx, s.db.RW(), row, req, []byte{}, mysqltype.DeploymentsStatusSkipped, reason)
				return err
			}, restate.WithName("insert skipped deployment")); err != nil {
				logger.Error("failed to insert skipped deployment", "app_id", row.AppID, "error", err)
			}
		}

		if !row.BuildSettingsAutoDeploy {
			logger.Info("skipping deployment: auto_deploy disabled",
				"app_id", row.AppID,
				"environment", row.EnvironmentSlug,
			)
			skipDeployment("Auto deploy is disabled for this environment.")
			continue
		}

		matched, matchErr := match.MatchWatchPaths(row.BuildSettingsWatchPaths, changedFiles)
		if matchErr != nil {
			// A broken pattern looks exactly like a valid miss, so the reason names
			// the pattern instead of blaming the changed files.
			logger.Warn("skipping deployment: invalid watch path",
				"app_id", row.AppID,
				"watch_paths", row.BuildSettingsWatchPaths,
				"error", matchErr,
			)
			skipDeployment(fault.UserFacingMessage(matchErr))
			continue
		}
		if !matched {
			logger.Info("skipping deployment: watch paths don't match changed files",
				"app_id", row.AppID,
				"watch_paths", row.BuildSettingsWatchPaths,
				"changed_files", changedFiles,
			)
			skipDeployment("Watch paths did not match any changed files.")
			continue
		}

		secretsBlob, marshalErr := buildSecretsBlob(envVarsByApp[row.AppID])
		if marshalErr != nil {
			logger.Error("failed to marshal secrets config", "appId", row.AppID, "error", marshalErr)
			continue
		}

		// Approval decision is independent of allowUnauthenticatedDeployments:
		// the flag only controls whether we reach out to GitHub (e.g. to post
		// the "awaiting authorization" commit status — see blockDeploymentForApproval).
		// Fork PRs run external code and must always be gated, even in dev.
		needsApproval := s.requiresApproval(req)

		status := mysqltype.DeploymentsStatusPending
		if needsApproval {
			status = mysqltype.DeploymentsStatusAwaitingApproval
		}

		deploymentID, insertErr := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return insertDeploymentRecord(runCtx, s.db.RW(), row, req, secretsBlob, status, "")
		}, restate.WithName("insert deployment"))
		if insertErr != nil {
			logger.Error("failed to insert deployment", "appId", row.AppID, "error", insertErr)
			continue
		}

		logger.Info("created deployment record",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", row.ProjectID,
			"app_id", row.AppID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
			"branch", req.GetBranch(),
			"environment", row.EnvironmentSlug,
			"needs_approval", needsApproval,
		)

		if needsApproval {
			if blockErr := s.blockDeploymentForApproval(ctx, req, row.ProjectWorkspaceID, row.ProjectID, row.ConnectionInstallationID, deploymentID); blockErr != nil {
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
					InstallationId: row.ConnectionInstallationID,
					Repository:     row.ConnectionRepositoryFullName,
					CommitSha:      req.GetAfter(),
					ContextPath:    row.BuildSettingsDockerContext,
					DockerfilePath: row.BuildSettingsDockerfile.String,
					BuildCommand:   row.BuildSettingsBuildCommand.String,
					PrNumber:       req.GetPrNumber(),
					ForkRepository: req.GetForkRepositoryFullName(),
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
		if persistErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.db.UpdateDeploymentInvocationID(runCtx, db.UpdateDeploymentInvocationIDParams{
				ID:           deploymentID,
				InvocationID: sql.NullString{Valid: true, String: invocationID},
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("persist invocation id")); persistErr != nil {
			// Without the invocation ID the deployment can never be
			// cancelled, so fail the handler and let Restate retry from
			// the journal (the Send above is journaled and not repeated).
			return nil, persistErr
		}

		logger.Info("deployment workflow started",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", row.ProjectID,
			"app_id", row.AppID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
			"invocation_id", invocationID,
		)

		// Cancelling superseded siblings is best-effort: the closure logs
		// and returns nil on failure. The RunVoid error itself must still
		// be propagated because it can carry Restate protocol signals
		// (suspension, cancellation), not just closure failures.
		if runErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			if cancelErr := s.dedup.CancelOlderSiblings(runCtx, dedup.Newer{
				ID:            deploymentID,
				AppID:         row.AppID,
				EnvironmentID: row.EnvironmentID,
				GitBranch:     req.GetBranch(),
				CreatedAt:     time.Now().UnixMilli(),
			}); cancelErr != nil {
				logger.Error("failed to cancel superseded siblings",
					"deployment_id", deploymentID,
					"error", cancelErr,
				)
			}
			return nil
		}, restate.WithName("cancel superseded siblings")); runErr != nil {
			return nil, runErr
		}
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
	req *hydrav1.HandlePushRequest,
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

// triggerReasonBytesMax mirrors the deployments.trigger_reason column width.
// The column counts characters, so a byte budget always fits.
const triggerReasonBytesMax = 512

// insertDeploymentRecord creates a deployment and its initial queued step in a single transaction.
func insertDeploymentRecord(
	ctx context.Context,
	rw *db.Replica,
	row db.ListRepoConnectionDeployContextsRow,
	req *hydrav1.HandlePushRequest,
	secretsBlob []byte,
	status mysqltype.DeploymentsStatus,
	triggerReason string,
) (string, error) {
	triggerReason = trimLength(triggerReason, triggerReasonBytesMax)
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	commitSHA := req.GetAfter()
	branch := req.GetBranch()
	commitMessage := req.GetCommitMessage()
	authorHandle := req.GetCommitAuthorHandle()
	authorAvatarURL := req.GetCommitAuthorAvatarUrl()
	commitTimestamp := req.GetCommitTimestamp()

	err := db.Tx(ctx, rw, func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.NewQueries(tx).InsertDeployment(txCtx, db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   row.ProjectWorkspaceID,
			ProjectID:                     row.ProjectID,
			AppID:                         row.AppID,
			EnvironmentID:                 row.EnvironmentID,
			Source:                        db.DeploymentsSourceGit,
			ImageRequested:                sql.NullString{Valid: false},
			SentinelConfig:                row.RuntimeSettingsSentinelConfig,
			EncryptedEnvironmentVariables: secretsBlob,
			Command:                       row.RuntimeSettingsCommand,
			Status:                        status,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false},
			GitCommitSha:                  sql.NullString{String: commitSHA, Valid: commitSHA != ""},
			GitBranch:                     sql.NullString{String: branch, Valid: branch != ""},
			GitCommitMessage:              sql.NullString{String: commitMessage, Valid: commitMessage != ""},
			GitCommitAuthorHandle:         sql.NullString{String: authorHandle, Valid: authorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: authorAvatarURL, Valid: authorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: commitTimestamp, Valid: commitTimestamp != 0},
			CpuMillicores:                 row.RuntimeSettingsCpuMillicores,
			MemoryMib:                     row.RuntimeSettingsMemoryMib,
			StorageMib:                    row.RuntimeSettingsStorageMib,
			Port:                          row.RuntimeSettingsPort,
			ShutdownSignal:                db.DeploymentsShutdownSignal(row.RuntimeSettingsShutdownSignal),
			UpstreamProtocol:              db.DeploymentsUpstreamProtocol(row.RuntimeSettingsUpstreamProtocol),
			Healthcheck:                   row.RuntimeSettingsHealthcheck,
			PrNumber:                      sql.NullInt64{Int64: req.GetPrNumber(), Valid: req.GetPrNumber() != 0},
			ForkRepositoryFullName:        sql.NullString{String: req.GetForkRepositoryFullName(), Valid: req.GetForkRepositoryFullName() != ""},
			DeploymentTrigger:             db.DeploymentsTriggerGithub,
			TriggeredBy:                   sql.NullString{String: req.GetSenderLogin(), Valid: req.GetSenderLogin() != ""},
			TriggerReason:                 sql.NullString{String: triggerReason, Valid: triggerReason != ""},
		}); txErr != nil {
			return txErr
		}

		return db.NewQueries(tx).InsertDeploymentStep(txCtx, db.InsertDeploymentStepParams{
			WorkspaceID:   row.ProjectWorkspaceID,
			ProjectID:     row.ProjectID,
			AppID:         row.AppID,
			EnvironmentID: row.EnvironmentID,
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

// trimLength truncates s to at most maxBytes bytes while preserving valid
// UTF-8: if the byte limit lands inside a multi-byte rune, the truncation
// happens at the previous rune boundary instead. MySQL strict mode rejects
// malformed UTF-8, and a single invalid watch path can exceed the column width.
func trimLength(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
