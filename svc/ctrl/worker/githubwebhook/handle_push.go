package githubwebhook

import (
	"database/sql"
	"os"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// HandlePush turns one GitHub push into one deployment per matching app.
//
// It decides what should happen to each app and hands the writing to
// DeployService.Create, which owns every deployment row. The sends are
// asynchronous: a push touching several apps must not have one slow app's
// GitHub lookup hold up the others, and the webhook has nothing to report back
// to GitHub that would need the answers.
//
// Because those sends can land out of order, every Create in one push carries
// the same ordering timestamp: the moment this handler received the push. That
// timestamp becomes the row's created_at, which is what sibling dedup and the
// supersede check compare. Without it a push that lands late would look newer
// than the commit that followed it and could cancel it.
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

	// Single query: connections + apps + projects + environments + build/runtime
	// settings. Picks the production environment for the default branch and
	// preview for everything else; a fork PR always goes to preview.
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
		buildSettings := row.AppBuildSetting

		switch {
		case !buildSettings.AutoDeploy:
			logger.Info("skipping deployment: auto_deploy disabled",
				"app_id", row.App.ID,
				"environment", row.Environment.Slug,
			)
			decisions = append(decisions, pushDecision{
				row:      row,
				decision: hydrav1.CreateDecision_CREATE_DECISION_SKIP,
			})

		case !match.MatchWatchPaths(buildSettings.WatchPaths, changedFiles):
			logger.Info("skipping deployment: watch paths don't match changed files",
				"app_id", row.App.ID,
				"watch_paths", buildSettings.WatchPaths,
				"changed_files", changedFiles,
			)
			decisions = append(decisions, pushDecision{
				row:      row,
				decision: hydrav1.CreateDecision_CREATE_DECISION_SKIP,
			})

		default:
			decisions = append(decisions, pushDecision{row: row, decision: matchedDecision})
		}
	}

	// Mint every id in one journaled step. uid.New outside a Run would produce
	// different ids on a retry, and the sends below are journaled individually:
	// a crash midway through the loop would then send a second create for the
	// apps it had already reached.
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
			ProjectId:   row.Project.ID,
			AppId:       row.App.ID,
			Environment: row.Environment.ID,
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
			Command:           nil,
			Decision:          d.decision,
			OrderingTimestamp: receivedAt.UnixMilli(),
			Trigger:           ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_GITHUB,
			TriggeredBy:       req.GetSenderLogin(),
			TriggerReason:     "",
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
			"project_id", row.Project.ID,
			"app_id", row.App.ID,
			"environment", row.Environment.Slug,
			"decision", d.decision.String(),
		)
	}

	return &hydrav1.HandlePushResponse{}, nil
}

// pushDecision pairs a matched app with what should happen to it.
type pushDecision struct {
	row      db.ListRepoConnectionDeployContextsRow
	decision hydrav1.CreateDecision
}

// eligibleContexts drops the apps whose workspace may not deploy.
//
// The gate runs here, before any GitHub call and before a single row is
// written, because a rejection has to be a successful no-op: a workspace that
// will never be eligible would otherwise fail this invocation on every push and
// stall the repository behind Restate's retries. Create runs the same gate for
// its own callers, so this is the short-circuit rather than the enforcement.
func (s *Service) eligibleContexts(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	contexts []db.ListRepoConnectionDeployContextsRow,
) ([]db.ListRepoConnectionDeployContextsRow, error) {
	entitlements := make(map[string]db.FindWorkspaceDeployEntitlementRow)
	eligible := make([]db.ListRepoConnectionDeployContextsRow, 0, len(contexts))

	for _, row := range contexts {
		project := row.Project
		app := row.App

		entitlement, ok := entitlements[project.WorkspaceID]
		if !ok {
			loaded, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindWorkspaceDeployEntitlementRow, error) {
				found, loadErr := s.db.FindWorkspaceDeployEntitlement(runCtx, project.WorkspaceID)
				if db.IsNotFound(loadErr) {
					return db.FindWorkspaceDeployEntitlementRow{
						Plan:           sql.NullString{},
						PlanOverride:   sql.NullString{},
						SpendSuspended: sql.NullBool{},
					}, nil
				}
				return found, loadErr
			}, restate.WithName("load workspace deploy entitlement "+project.WorkspaceID))
			if err != nil {
				return nil, err
			}
			entitlement = loaded
			entitlements[project.WorkspaceID] = entitlement
		}

		if !deploygate.Entitled(entitlement.Plan, entitlement.PlanOverride) {
			if s.enforceDeployGate {
				logger.Info("skipping deployment: workspace has no Compute plan",
					"event", "deploy_gate.blocked",
					"reason", "no_plan",
					"workspace_id", project.WorkspaceID,
					"project_id", project.ID,
					"app_id", app.ID,
					"delivery_id", req.GetDeliveryId(),
				)
				continue
			}
			logger.Warn("deploy gate would block GitHub deployment",
				"event", "deploy_gate.would_block",
				"workspaceId", project.WorkspaceID,
				"projectId", project.ID,
				"appId", app.ID,
			)
		}

		if entitlement.SpendSuspended.Bool {
			logger.Info("skipping deployment: workspace is spend suspended",
				"event", "deploy_gate.blocked",
				"reason", "spend_suspended",
				"workspace_id", project.WorkspaceID,
				"project_id", project.ID,
				"app_id", app.ID,
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
// Webhook payloads do not always include per-commit file lists: a fork PR
// arrives through the pull_request event, which has no commits, and a
// created-branch push pointing at an already-reachable commit arrives with an
// empty commits array. Without the diff, watch-path matching would skip apps
// that should have deployed, so an empty list is worth a round trip.
//
// A failed fetch is not fatal. It falls back to no files, which matches nothing
// and skips the deployment, and that is the safer of the two errors: a skipped
// row is visible and re-pushable, a wrongly built one is already running.
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
