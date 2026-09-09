package githubwebhook

import (
	"database/sql"
	"os"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// HandlePush processes a GitHub push event: it looks up the repo connections,
// matches each app's watch paths against the changed files, then calls
// DeployService.Create once per app. Create owns the workspace entitlement gate.
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
	isForkPR := int64(0)
	if req.GetIsForkPr() {
		isForkPR = 1
	}

	// The default branch resolves to production, any other branch to preview,
	// and a fork PR always to preview.
	contexts, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListRepoConnectionDeployContextsRow, error) {
		return s.db.ListRepoConnectionDeployContexts(runCtx, db.ListRepoConnectionDeployContextsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
			Branch:         sql.NullString{String: branch, Valid: branch != ""},
			IsForkPr:       isForkPR,
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

	// Ids are minted inside restate.Run so a replay reuses the same ones.
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

	// Fire every Create first and await them in a second loop, so the apps'
	// creates run in parallel instead of one after another.
	pending := make([]pendingCreate, 0, len(contexts))

	for i, row := range contexts {
		deploymentID := ids[i]

		// A skip still goes through Create so the dashboard shows the commit
		// arrived, with this reason on the row.
		skipDeployment := func(reason string) {
			pending = append(pending, pendingCreate{
				deploymentID:  deploymentID,
				appID:         row.AppID,
				environmentID: row.EnvironmentID,
				decision:      hydrav1.CreateDecision_CREATE_DECISION_SKIP,
				reason:        reason,
				future: s.startCreate(ctx, createArgs{
					deploymentID: deploymentID,
					row:          row,
					req:          req,
					decision:     hydrav1.CreateDecision_CREATE_DECISION_SKIP,
					reason:       reason,
				}),
			})
		}

		if !row.BuildSettingsAutoDeploy {
			skipDeployment("Auto deploy is disabled for this environment.")
			continue
		}

		matched, matchErr := match.MatchWatchPaths(row.BuildSettingsWatchPaths, changedFiles)
		if matchErr != nil {
			// A broken pattern looks exactly like a valid miss, so the reason names
			// the pattern instead of blaming the changed files.
			skipDeployment(fault.UserFacingMessage(matchErr))
			continue
		}
		if !matched {
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
			deploymentID:  deploymentID,
			appID:         row.AppID,
			environmentID: row.EnvironmentID,
			decision:      decision,
			reason:        "",
			future: s.startCreate(ctx, createArgs{
				deploymentID: deploymentID,
				row:          row,
				req:          req,
				decision:     decision,
				reason:       "",
			}),
		})
	}

	// Awaiting here is what orders pushes: this repository's object stays held
	// until every row is written, so the next push gets a later created_at and
	// supersedes these rows instead of the other way round.
	//
	// Only a terminal error from Create lands here, which is a bug in Create.
	// Returning it would make Restate retry this handler forever and block every
	// later push to the repository, so it is logged and the push succeeds.
	for _, create := range pending {
		resp, err := create.future.Response()
		if err != nil {
			logger.Error(
				"deployment create failed",
				"deployment_id", create.deploymentID,
				"delivery_id", req.GetDeliveryId(),
				"app_id", create.appID,
				"error", err,
			)
			continue
		}

		logger.Info(
			"deployment create finished",
			"deployment_id", create.deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
			"branch", req.GetBranch(),
			"app_id", create.appID,
			"environment_id", create.environmentID,
			"decision", create.decision.String(),
			"reason", create.reason,
			"outcome", resp.GetOutcome().String(),
		)

		if resp.GetOutcome() != hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED {
			s.postRejectedStatus(ctx, req, create.deploymentID, resp.GetDetail())
		}
	}
	return &hydrav1.HandlePushResponse{}, nil
}

// commitStatusDescriptionMax is the limit the Status API enforces. It is not in
// the public docs; a longer description fails with "description is too long
// (maximum is 140 characters)", see github.com/zalando/zappr/issues/378
const commitStatusDescriptionMax = 140

// postRejectedStatus puts the worker's reason on the pushed commit. A rejected
// create writes no row, so without this the push disappears without a trace
// anywhere the developer looks. GitHub errors are retried for a bounded time,
// then logged and dropped: the reason is already in the log above.
func (s *Service) postRejectedStatus(ctx restate.ObjectContext, req *hydrav1.HandlePushRequest, deploymentID, detail string) {
	if s.allowUnauthenticatedDeployments {
		return
	}

	err := restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return s.github.CreateCommitStatus(
			req.GetInstallationId(),
			req.GetRepositoryFullName(),
			req.GetAfter(),
			"error",
			"",
			truncate(detail, commitStatusDescriptionMax),
			githubclient.DeployRejectedContext,
		)
	}, restate.WithName("create commit status for rejected create"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Error(
			"failed to post rejected commit status",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"error", err,
		)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

type pendingCreate struct {
	deploymentID  string
	appID         string
	environmentID string
	decision      hydrav1.CreateDecision
	reason        string
	future        restate.ResponseFuture[*hydrav1.DeployCreateResponse]
}

type createArgs struct {
	deploymentID string
	row          db.ListRepoConnectionDeployContextsRow
	req          *hydrav1.HandlePushRequest
	decision     hydrav1.CreateDecision
	reason       string
}

// startCreate calls DeployService.Create for one app and returns the future.
//
// Why RequestFuture and not Send or Request:
//
// Supersede logic is decided by created_at. Create inserts the row then cancels
// siblings with a smaller timestamp, so the newer push has to insert later.
// This used to be guaranteed because HandlePush owned the row insert, but that
// part moved to Create. From GitHub's point of view nothing changed, the
// webhook handler calls HandlePush async anyway.
//
// If we Send, Push A returns as soon as the send is journaled, Push B starts,
// and Create A and Create B race. If B lands first, A supersedes B and we
// deploy the older commit. Awaiting keeps Push A alive until every row is
// stamped, so B can't slip in.
//
// Plain Request would fix ordering too but runs the apps one after another.
// RequestFuture lets us fire every app's create first and await them after, so
// they still run concurrently like the old Send.
func (s *Service) startCreate(
	ctx restate.ObjectContext,
	args createArgs,
) restate.ResponseFuture[*hydrav1.DeployCreateResponse] {
	row := args.row
	req := args.req

	event := "push"
	if req.GetIsForkPr() {
		event = "pull_request"
	}

	return hydrav1.NewDeployServiceClient(ctx, args.deploymentID).Create().RequestFuture(&hydrav1.DeployCreateRequest{
		ProjectId:     row.ProjectID,
		AppId:         row.AppID,
		EnvironmentId: row.EnvironmentID,
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
		Decision:      args.decision,
		Trigger:       ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_GITHUB,
		TriggeredBy:   req.GetSenderLogin(),
		TriggerReason: args.reason,
		Actor: &ctrlv1.ActorInfo{
			Id:        req.GetSenderLogin(),
			Name:      req.GetSenderLogin(),
			Type:      ctrlv1.ActorType_ACTOR_TYPE_GITHUB,
			RemoteIp:  "",
			UserAgent: "",
			Meta: map[string]string{
				"delivery_id": req.GetDeliveryId(),
				"event":       event,
				"repository":  req.GetRepositoryFullName(),
			},
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
