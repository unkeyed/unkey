package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// AuthorizeDeployment authorizes a deployment that is awaiting approval.
// It looks up the deployment by ID, verifies it is in awaiting_approval status,
// updates the status to pending, and triggers the deploy workflow.
func (s *Service) AuthorizeDeployment(ctx context.Context, req *connect.Request[ctrlv1.AuthorizeDeploymentRequest]) (*connect.Response[ctrlv1.AuthorizeDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()

	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}

	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find deployment: %w", err))
	}

	if deployment.Status != db.DeploymentsStatusAwaitingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s is not awaiting approval (current status: %s)", deploymentID, deployment.Status))
	}

	// Update status to pending so the deployment workflow can proceed
	err = db.Query.UpdateDeploymentStatus(ctx, s.db.RW(), db.UpdateDeploymentStatusParams{
		ID:     deploymentID,
		Status: db.DeploymentsStatusPending,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment status: %w", err))
	}

	// Look up build settings and repo connection to construct the deploy request
	buildSetting, err := db.Query.FindAppBuildSettingByAppEnv(ctx, s.db.RO(), db.FindAppBuildSettingByAppEnvParams{
		AppID:         deployment.AppID,
		EnvironmentID: deployment.EnvironmentID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find build settings: %w", err))
	}

	repoConn, err := db.Query.FindGithubRepoConnectionByProjectId(ctx, s.db.RO(), deployment.ProjectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find repo connection: %w", err))
	}

	commitSHA := ""
	if deployment.GitCommitSha.Valid {
		commitSHA = deployment.GitCommitSha.String
	}

	deployReq := &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		Source: &hydrav1.DeployRequest_Git{
			Git: &hydrav1.GitSource{
				InstallationId: repoConn.InstallationID,
				Repository:     repoConn.RepositoryFullName,
				CommitSha:      commitSHA,
				ContextPath:    buildSetting.DockerContext,
				DockerfilePath: buildSetting.Dockerfile,
			},
		},
	}

	_, sendErr := s.deploymentClient(deployment.ProjectID).Deploy().Send(ctx, deployReq)
	if sendErr != nil {
		logger.Error("failed to trigger deploy workflow after authorization",
			"deployment_id", deploymentID,
			"error", sendErr,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deploy workflow: %w", sendErr))
	}

	// Update commit status on GitHub
	if commitSHA != "" {
		if statusErr := s.github.CreateCommitStatus(
			repoConn.InstallationID,
			repoConn.RepositoryFullName,
			commitSHA,
			"success",
			"",
			"Deployment authorized and started",
			"Unkey Deploy Authorization",
		); statusErr != nil {
			logger.Error("failed to update commit status to success", "error", statusErr)
		}
	}

	logger.Info("deployment authorized and workflow triggered",
		"deployment_id", deploymentID,
		"project_id", deployment.ProjectID,
	)

	return connect.NewResponse(&ctrlv1.AuthorizeDeploymentResponse{}), nil
}
