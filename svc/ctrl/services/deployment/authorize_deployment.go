package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deployutil"
)

// AuthorizeDeployment authorizes a deployment for an external contributor's push.
// It fetches the current HEAD of the branch from GitHub, re-derives all matching
// deploy contexts, creates deployment records, and fires deploy workflows.
func (s *Service) AuthorizeDeployment(ctx context.Context, req *connect.Request[ctrlv1.AuthorizeDeploymentRequest]) (*connect.Response[ctrlv1.AuthorizeDeploymentResponse], error) {
	projectID := req.Msg.GetProjectId()
	branch := req.Msg.GetBranch()

	if projectID == "" || branch == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and branch are required"))
	}

	repoConn, err := db.Query.FindGithubRepoConnectionByProjectId(ctx, s.db.RO(), projectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no GitHub connection found for project %s", projectID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find repo connection: %w", err))
	}

	headCommit, err := s.github.GetBranchHeadCommit(repoConn.InstallationID, repoConn.RepositoryFullName, branch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch branch HEAD from GitHub: %w", err))
	}

	contexts, err := db.Query.ListRepoConnectionDeployContexts(ctx, s.db.RO(), db.ListRepoConnectionDeployContextsParams{
		InstallationID: repoConn.InstallationID,
		RepositoryID:   repoConn.RepositoryID,
		Branch:         branch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list deploy contexts: %w", err))
	}

	if len(contexts) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no deploy contexts found for project %s branch %s", projectID, branch))
	}

	allEnvVars, err := db.Query.ListEnvVarsForRepoConnections(ctx, s.db.RO(), db.ListEnvVarsForRepoConnectionsParams{
		InstallationID: repoConn.InstallationID,
		RepositoryID:   repoConn.RepositoryID,
		Branch:         branch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list env vars: %w", err))
	}

	envVarsByApp := deployutil.GroupEnvVarsByApp(allEnvVars)

	commit := deployutil.GitCommitInfo{
		SHA:             headCommit.SHA,
		Branch:          branch,
		Message:         headCommit.Message,
		AuthorHandle:    headCommit.AuthorHandle,
		AuthorAvatarURL: headCommit.AuthorAvatarURL,
		Timestamp:       headCommit.Timestamp.UnixMilli(),
	}

	for _, row := range contexts {
		secretsBlob, marshalErr := deployutil.BuildSecretsBlob(envVarsByApp[row.App.ID])
		if marshalErr != nil {
			logger.Error("failed to marshal secrets config", "appId", row.App.ID, "error", marshalErr)
			continue
		}

		deploymentID, insertErr := deployutil.InsertDeploymentRecord(ctx, s.db.RW(), row, commit, secretsBlob, db.DeploymentsStatusPending)
		if insertErr != nil {
			logger.Error("failed to insert deployment for authorization", "appId", row.App.ID, "error", insertErr)
			continue
		}

		deployReq := deployutil.BuildDeployRequest(deploymentID, row, headCommit.SHA)
		_, sendErr := s.deploymentClient(row.Project.ID).Deploy().Send(ctx, deployReq)
		if sendErr != nil {
			logger.Error("failed to trigger deploy workflow after authorization",
				"deployment_id", deploymentID,
				"error", sendErr,
			)
			continue
		}

		logger.Info("deployment authorized and workflow triggered",
			"deployment_id", deploymentID,
			"project_id", row.Project.ID,
			"app_id", row.App.ID,
			"branch", branch,
			"commit_sha", headCommit.SHA,
		)
	}

	// Mark the PR check run as green now that the deployment is authorized
	checkRuns, listErr := s.github.ListCheckRunsForRef(repoConn.InstallationID, repoConn.RepositoryFullName, headCommit.SHA, "Unkey Deploy Authorization")
	if listErr == nil {
		for _, cr := range checkRuns {
			if updateErr := s.github.UpdateCheckRun(
				repoConn.InstallationID,
				repoConn.RepositoryFullName,
				cr.ID,
				"completed",
				"success",
				"Deployment authorized",
				"Deployment authorized and started by a project member.",
			); updateErr != nil {
				logger.Error("failed to update check run to success", "check_run_id", cr.ID, "error", updateErr)
			}
		}
	}

	return connect.NewResponse(&ctrlv1.AuthorizeDeploymentResponse{}), nil
}
