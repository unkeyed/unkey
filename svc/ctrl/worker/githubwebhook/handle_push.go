package githubwebhook

import (
	"os"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deployutil"
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

	envVarsByApp := deployutil.GroupEnvVarsByApp(allEnvVars)

	commit := deployutil.GitCommitInfo{
		SHA:             req.GetAfter(),
		Branch:          branch,
		Message:         req.GetCommitMessage(),
		AuthorHandle:    req.GetCommitAuthorHandle(),
		AuthorAvatarURL: req.GetCommitAuthorAvatarUrl(),
		Timestamp:       req.GetCommitTimestamp(),
	}

	for _, row := range contexts {
		project := row.Project
		env := row.Environment
		app := row.App
		repo := row.GithubRepoConnection

		secretsBlob, marshalErr := deployutil.BuildSecretsBlob(envVarsByApp[app.ID])
		if marshalErr != nil {
			logger.Error("failed to marshal secrets config", "appId", app.ID, "error", marshalErr)
			continue
		}

		needsApproval := false
		if app.DeploymentProtection {
			needsApproval = s.requiresApproval(ctx, req, repo)
		}

		if needsApproval {
			if blockErr := s.blockDeploymentForApproval(ctx, req, project, env, app, repo, branch); blockErr != nil {
				return nil, blockErr
			}
			continue
		}

		deploymentID, insertErr := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return deployutil.InsertDeploymentRecord(runCtx, s.db.RW(), row, commit, secretsBlob, db.DeploymentsStatusPending)
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
		)

		deployClient := hydrav1.NewDeployServiceClient(ctx, app.WorkspaceID)
		deployClient.Deploy().Send(deployutil.BuildDeployRequest(deploymentID, row, req.GetAfter()))

		logger.Info("deployment workflow started",
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
//
// Set FORCE_DEPLOYMENT_APPROVAL=true to bypass collaborator checks and always
// require approval. This is useful for testing the approval flow locally.
func (s *Service) requiresApproval(
	ctx restate.ObjectContext,
	req *hydrav1.HandlePushRequest,
	repo db.GithubRepoConnection,
) bool {
	if os.Getenv("FORCE_DEPLOYMENT_APPROVAL") == "true" {
		logger.Info("FORCE_DEPLOYMENT_APPROVAL is set, requiring approval",
			"sender", req.GetSenderLogin(),
		)
		return true
	}

	senderLogin := req.GetSenderLogin()

	// Bot accounts are trusted (GitHub controls the [bot] suffix)
	if strings.HasSuffix(senderLogin, "[bot]") {
		return false
	}

	// No sender info — fail closed, require approval
	if senderLogin == "" {
		logger.Info("no sender login in push event, requiring approval")
		return true
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
