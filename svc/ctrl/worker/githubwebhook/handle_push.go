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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// HandlePush processes a GitHub push event: it looks up the repo connections,
// gates them on the workspace entitlement and each app's watch paths, then
// calls DeployService.Create once per app.
func (s *Service) HandlePush(ctx restate.ObjectContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	logger.Info(
		"handling GitHub push in Restate",
		"delivery_id", req.GetDeliveryId(),
		"repository", req.GetRepositoryFullName(),
		"branch", req.GetBranch(),
		"commit_sha", req.GetAfter(),
		"sender_login", req.GetSenderLogin(),
	)

	branch := req.GetBranch()

	// The default branch resolves to production, any other branch to preview,
	// and a fork PR always to preview.
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
		logger.Info(
			"no deploy contexts found",
			"installation_id", req.GetInstallationId(),
			"repository_id", req.GetRepositoryId(),
			"branch", req.GetBranch(),
		)
		return &hydrav1.HandlePushResponse{}, nil
	}

	// Gate before calling GitHub or writing even a skipped row: an ineligible
	// workspace is a successful no-op, not something Restate should retry.
	workspaceIDs := make([]string, 0, len(contexts))
	seenWorkspace := make(map[string]bool, len(contexts))
	for _, row := range contexts {
		if !seenWorkspace[row.ProjectWorkspaceID] {
			seenWorkspace[row.ProjectWorkspaceID] = true
			workspaceIDs = append(workspaceIDs, row.ProjectWorkspaceID)
		}
	}

	entitlementRows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListWorkspaceDeployEntitlementsRow, error) {
		return s.db.ListWorkspaceDeployEntitlements(runCtx, workspaceIDs)
	}, restate.WithName("load workspace deploy entitlements"))
	if err != nil {
		return nil, err
	}

	entitlements := make(map[string]db.ListWorkspaceDeployEntitlementsRow, len(entitlementRows))
	for _, entitlement := range entitlementRows {
		entitlements[entitlement.WorkspaceID] = entitlement
	}

	eligibleContexts := make([]db.ListRepoConnectionDeployContextsRow, 0, len(contexts))
	for _, row := range contexts {
		// A workspace missing from the map reads as no plan and not suspended,
		// the same as an unbilled one.
		entitlement := entitlements[row.ProjectWorkspaceID]

		if !deploygate.Entitled(entitlement.Plan, entitlement.PlanOverride) {
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
		if entitlement.SpendSuspended.Bool {
			logger.Info(
				"skipping deployment: workspace is spend suspended",
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

	// Webhook payloads don't always include per-commit file lists:
	//   - Fork PRs come through the pull_request webhook which has no commits.
	//   - Created-branch pushes pointing at an already-reachable commit arrive
	//     with an empty commits array.
	// When files aren't available, fetch from the GitHub API so watch-path
	// matching doesn't skip deploys for lack of a diff.
	changedFiles := req.GetChangedFiles()
	if len(changedFiles) == 0 && req.GetAfter() != "" && !s.allowUnauthenticatedDeployments {
		logger.Info(
			"fetching commit files from GitHub",
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
			logger.Error(
				"failed to list commit files, proceeding with empty changed files",
				"commit_sha", req.GetAfter(),
				"error", filesErr,
			)
		} else {
			logger.Info(
				"fetched commit files",
				"commit_sha", req.GetAfter(),
				"changed_files", files,
			)
			changedFiles = files
		}
	}

	// uid.New is random, so the ids are journaled, and in one step rather than
	// one per app.
	ids, err := restate.Run(ctx, func(_ restate.RunContext) ([]string, error) {
		minted := make([]string, len(contexts))
		for i := range minted {
			minted[i] = uid.New(uid.DeploymentPrefix)
		}
		return minted, nil
	}, restate.WithName("mint deployment ids"))
	if err != nil {
		return nil, err
	}

	// All creates are started before any is awaited so their GitHub lookups
	// overlap.
	pending := make([]pendingCreate, 0, len(contexts))

	for i, row := range contexts {
		deploymentID := ids[i]

		// The dashboard shows this reason on the skipped deployment.
		skipDeployment := func(reason string) {
			pending = append(pending, pendingCreate{
				deploymentID: deploymentID,
				appID:        row.AppID,
				future: s.requestCreate(ctx, deploymentID, row, req,
					hydrav1.CreateDecision_CREATE_DECISION_SKIP, reason),
			})
		}

		if !row.BuildSettingsAutoDeploy {
			logger.Info(
				"skipping deployment: auto_deploy disabled",
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
			logger.Warn(
				"skipping deployment: invalid watch path",
				"app_id", row.AppID,
				"watch_paths", row.BuildSettingsWatchPaths,
				"error", matchErr,
			)
			skipDeployment(fault.UserFacingMessage(matchErr))
			continue
		}
		if !matched {
			logger.Info(
				"skipping deployment: watch paths don't match changed files",
				"app_id", row.AppID,
				"watch_paths", row.BuildSettingsWatchPaths,
				"changed_files", changedFiles,
			)
			skipDeployment("Watch paths did not match any changed files.")
			continue
		}

		// Approval is independent of allowUnauthenticatedDeployments: that flag only
		// decides whether Unkey talks to GitHub. Fork PRs run external code and are
		// gated even in local development.
		decision := hydrav1.CreateDecision_CREATE_DECISION_DEPLOY
		if s.requiresApproval(req) {
			decision = hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL
		}

		pending = append(pending, pendingCreate{
			deploymentID: deploymentID,
			appID:        row.AppID,
			future:       s.requestCreate(ctx, deploymentID, row, req, decision, ""),
		})
	}

	for _, create := range pending {
		resp, err := create.future.Response()
		if err != nil {
			return nil, err
		}

		// A rejected create writes no row, so this log is its only trace.
		if resp.GetOutcome() == hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED {
			logger.Warn(
				"deployment create rejected",
				"deployment_id", create.deploymentID,
				"delivery_id", req.GetDeliveryId(),
				"app_id", create.appID,
				"reason", resp.GetRejectionReason().String(),
			)
			continue
		}

		logger.Info(
			"deployment created",
			"deployment_id", create.deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"app_id", create.appID,
		)
	}

	return &hydrav1.HandlePushResponse{}, nil
}

type pendingCreate struct {
	deploymentID string
	appID        string
	future       restate.ResponseFuture[*hydrav1.DeployCreateResponse]
}

// requestCreate asks DeployService.Create for one app's deployment. It is a
// Request rather than a one-way Send: Create stamps created_at, which orders
// siblings, so awaiting it inside this per-repository object keeps two pushes
// to one repository in order.
func (s *Service) requestCreate(
	ctx restate.ObjectContext,
	deploymentID string,
	row db.ListRepoConnectionDeployContextsRow,
	req *hydrav1.HandlePushRequest,
	decision hydrav1.CreateDecision,
	reason string,
) restate.ResponseFuture[*hydrav1.DeployCreateResponse] {
	return hydrav1.NewDeployServiceClient(ctx, deploymentID).Create().RequestFuture(&hydrav1.DeployCreateRequest{
		ProjectId:   row.ProjectID,
		AppId:       row.AppID,
		Environment: row.EnvironmentID,
		Source: &hydrav1.DeployCreateRequest_Git{
			Git: &hydrav1.CreateGitSource{
				Commit: &ctrlv1.GitCommitInfo{
					CommitSha:       req.GetAfter(),
					Branch:          req.GetBranch(),
					CommitMessage:   req.GetCommitMessage(),
					AuthorHandle:    req.GetCommitAuthorHandle(),
					AuthorAvatarUrl: req.GetCommitAuthorAvatarUrl(),
					Timestamp:       req.GetCommitTimestamp(),
					ForkRepository:  req.GetForkRepositoryFullName(),
				},
				PrNumber: req.GetPrNumber(),
			},
		},
		Decision:      decision,
		Trigger:       ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_GITHUB,
		TriggeredBy:   req.GetSenderLogin(),
		TriggerReason: reason,
		Actor: &ctrlv1.ActorInfo{
			Id:        req.GetSenderLogin(),
			Name:      req.GetSenderLogin(),
			Type:      ctrlv1.ActorType_ACTOR_TYPE_GITHUB,
			RemoteIp:  "",
			UserAgent: "",
			Meta:      nil,
		},
	})
}

// requiresApproval is true for a fork PR, whose code comes from someone without
// write access. A direct push is already authorized by GitHub.
// FORCE_DEPLOYMENT_APPROVAL=true gates every push, for testing the flow locally.
func (s *Service) requiresApproval(
	req *hydrav1.HandlePushRequest,
) bool {
	if os.Getenv("FORCE_DEPLOYMENT_APPROVAL") == "true" {
		logger.Info(
			"FORCE_DEPLOYMENT_APPROVAL is set, requiring approval",
			"sender", req.GetSenderLogin(),
		)
		return true
	}

	if req.GetIsForkPr() {
		logger.Info(
			"fork PR deployment requires approval",
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
