package githubwebhook

import (
	"database/sql"
	"os"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// HandlePush turns one GitHub push into one deployment per matching app.
//
// It decides what happens to each app and hands the writing to
// DeployService.Create, which owns every deployment row. The sends are one-way,
// so a slow Create for one app does not hold up the others.
//
// Those sends can land out of order, so every Create in one push carries the
// same push_received_at, the moment this handler received the push. That
// timestamp becomes the row's created_at, which sibling dedup and the supersede
// check compare. Without it a push that lands late would look newer than the
// commit that followed it and could cancel it.
func (s *Service) HandlePush(ctx restate.ObjectContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	logger.Info("handling GitHub push in Restate",
		"delivery_id", req.GetDeliveryId(),
		"repository", req.GetRepositoryFullName(),
		"branch", req.GetBranch(),
		"commit_sha", req.GetAfter(),
		"sender_login", req.GetSenderLogin(),
	)

	receivedAt, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}

	branch := req.GetBranch()

	// The query picks the production environment for the default branch and
	// preview for every other branch. A fork PR always resolves to preview.
	contexts, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListRepoConnectionDeployContextsRow, error) {
		return s.db.ListRepoConnectionDeployContexts(runCtx, db.ListRepoConnectionDeployContextsParams{
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

	contexts, err = s.eligibleContexts(ctx, req, contexts)
	if err != nil {
		return nil, err
	}
	if len(contexts) == 0 {
		return &hydrav1.HandlePushResponse{}, nil
	}

	changedFiles, err := s.resolveChangedFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	// Approval is independent of allowUnauthenticatedDeployments: that flag only
	// decides whether Unkey talks to GitHub. Fork PRs run external code and are
	// gated even in local development.
	matchedDecision := hydrav1.CreateDecision_CREATE_DECISION_DEPLOY
	if s.requiresApproval(req) {
		matchedDecision = hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL
	}

	decisions := make([]pushDecision, 0, len(contexts))
	for _, row := range contexts {
		skip := func(reason string) {
			decisions = append(decisions, pushDecision{
				row:      row,
				decision: hydrav1.CreateDecision_CREATE_DECISION_SKIP,
				reason:   reason,
			})
		}

		if !row.BuildSettingsAutoDeploy {
			logger.Info("skipping deployment: auto_deploy disabled",
				"app_id", row.AppID,
				"environment", row.EnvironmentSlug,
			)
			skip("Auto deploy is disabled for this environment.")
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
			skip(fault.UserFacingMessage(matchErr))
			continue
		}
		if !matched {
			logger.Info("skipping deployment: watch paths don't match changed files",
				"app_id", row.AppID,
				"watch_paths", row.BuildSettingsWatchPaths,
				"changed_files", changedFiles,
			)
			skip("Watch paths did not match any changed files.")
			continue
		}

		decisions = append(decisions, pushDecision{row: row, decision: matchedDecision, reason: ""})
	}

	// uid.New is not deterministic, so the ids have to come from a journaled
	// step. A retry replays the recorded ids and addresses the same Create
	// objects instead of a fresh set.
	ids, err := restate.Run(ctx, func(_ restate.RunContext) ([]string, error) {
		minted := make([]string, len(decisions))
		for i := range minted {
			minted[i] = uid.New(uid.DeploymentPrefix)
		}
		return minted, nil
	}, restate.WithName("mint deployment ids"))
	if err != nil {
		return nil, err
	}

	for i, d := range decisions {
		deploymentID := ids[i]
		row := d.row

		hydrav1.NewDeployServiceClient(ctx, deploymentID).Create().Send(&hydrav1.DeployCreateRequest{
			ProjectId:   row.ProjectID,
			AppId:       row.AppID,
			Environment: row.EnvironmentID,
			Source: &hydrav1.DeployCreateRequest_Git{
				Git: &hydrav1.CreateGitSource{
					Commit: &ctrlv1.GitCommitInfo{
						CommitSha:       req.GetAfter(),
						Branch:          branch,
						CommitMessage:   req.GetCommitMessage(),
						AuthorHandle:    req.GetCommitAuthorHandle(),
						AuthorAvatarUrl: req.GetCommitAuthorAvatarUrl(),
						Timestamp:       req.GetCommitTimestamp(),
						ForkRepository:  req.GetForkRepositoryFullName(),
					},
					PrNumber: req.GetPrNumber(),
				},
			},
			Command:        nil,
			Decision:       d.decision,
			PushReceivedAt: receivedAt.UnixMilli(),
			Trigger:        ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_GITHUB,
			TriggeredBy:    req.GetSenderLogin(),
			TriggerReason:  d.reason,
			Actor: &ctrlv1.ActorInfo{
				Id:        req.GetSenderLogin(),
				Name:      req.GetSenderLogin(),
				Type:      ctrlv1.ActorType_ACTOR_TYPE_GITHUB,
				RemoteIp:  "",
				UserAgent: "",
				Meta:      nil,
			},
		})

		logger.Info("requested deployment create",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", row.ProjectID,
			"app_id", row.AppID,
			"environment", row.EnvironmentSlug,
			"decision", d.decision.String(),
		)
	}

	return &hydrav1.HandlePushResponse{}, nil
}

// pushDecision pairs a matched app with what should happen to it.
type pushDecision struct {
	row      db.ListRepoConnectionDeployContextsRow
	decision hydrav1.CreateDecision
	// reason is empty for a deploy and always set for a skip. The dashboard
	// reads it off the row, so a push that builds nothing still says why.
	reason string
}

// eligibleContexts drops the apps whose workspace may not deploy.
//
// The gate runs before any GitHub call and before any row is written, because a
// rejection has to be a successful no-op. Failing the invocation instead would
// leave a permanently ineligible workspace retrying on every push, and the
// virtual object key would stall every later push for that repository. Create
// runs the same gate, so this is a short-circuit and not the enforcement point.
func (s *Service) eligibleContexts(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	contexts []db.ListRepoConnectionDeployContextsRow,
) ([]db.ListRepoConnectionDeployContextsRow, error) {
	entitlements := make(map[string]db.FindWorkspaceDeployEntitlementRow)
	eligible := make([]db.ListRepoConnectionDeployContextsRow, 0, len(contexts))

	for _, row := range contexts {
		entitlement, ok := entitlements[row.ProjectWorkspaceID]
		if !ok {
			loaded, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindWorkspaceDeployEntitlementRow, error) {
				found, loadErr := s.db.FindWorkspaceDeployEntitlement(runCtx, row.ProjectWorkspaceID)
				if db.IsNotFound(loadErr) {
					return db.FindWorkspaceDeployEntitlementRow{
						Plan:           sql.NullString{},
						PlanOverride:   sql.NullString{},
						SpendSuspended: sql.NullBool{},
					}, nil
				}
				return found, loadErr
			}, restate.WithName("load workspace deploy entitlement "+row.ProjectWorkspaceID))
			if err != nil {
				return nil, err
			}
			entitlement = loaded
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

		eligible = append(eligible, row)
	}

	return eligible, nil
}

// resolveChangedFiles returns the files this push touched, fetching them from
// GitHub when the payload carries none.
//
// Webhook payloads do not always include per-commit file lists. A fork PR
// arrives through the pull_request event, which has no commits, and a
// created-branch push pointing at an already-reachable commit arrives with an
// empty commits array. Without the diff, watch-path matching would skip apps
// that should have deployed.
//
// A failed fetch is not fatal. It falls back to no files, which matches nothing
// and skips the deployment. A skipped row stays visible and can be re-pushed,
// while a wrongly built one is already running.
func (s *Service) resolveChangedFiles(ctx restate.ObjectContext, req *hydrav1.HandlePushRequest) ([]string, error) {
	changedFiles := req.GetChangedFiles()
	if len(changedFiles) > 0 || req.GetAfter() == "" || s.allowUnauthenticatedDeployments {
		return changedFiles, nil
	}

	logger.Info("fetching commit files from GitHub",
		"commit_sha", req.GetAfter(),
		"repo", req.GetRepositoryFullName(),
		"installation_id", req.GetInstallationId(),
		"is_fork_pr", req.GetIsForkPr(),
	)

	files, err := restate.Run(ctx, func(_ restate.RunContext) ([]string, error) {
		return s.github.ListCommitFiles(
			req.GetInstallationId(),
			req.GetRepositoryFullName(),
			req.GetAfter(),
		)
	}, restate.WithName("list commit files"))
	if err != nil {
		logger.Error("failed to list commit files, proceeding with empty changed files",
			"commit_sha", req.GetAfter(),
			"error", err,
		)
		return nil, nil
	}

	logger.Info("fetched commit files",
		"commit_sha", req.GetAfter(),
		"changed_files", files,
	)
	return files, nil
}

// requiresApproval reports whether a push needs a project member's approval.
//
// Fork PRs always do: they run code from outside the repository. A non-fork push
// does not, because GitHub already verified the pusher has write access, and
// anyone who can push can deploy.
//
// Set FORCE_DEPLOYMENT_APPROVAL=true to gate every push, which is how the
// approval flow is exercised locally.
func (s *Service) requiresApproval(req *hydrav1.HandlePushRequest) bool {
	if os.Getenv("FORCE_DEPLOYMENT_APPROVAL") == "true" {
		logger.Info("FORCE_DEPLOYMENT_APPROVAL is set, requiring approval",
			"sender", req.GetSenderLogin(),
		)
		return true
	}

	if req.GetIsForkPr() {
		logger.Info("fork PR deployment requires approval",
			"sender", req.GetSenderLogin(),
		)
		return true
	}

	return false
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
