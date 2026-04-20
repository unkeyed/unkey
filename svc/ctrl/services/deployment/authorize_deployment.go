package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// AuthorizeDeployment authorizes a deployment that is awaiting approval.
// It looks up the deployment by ID, verifies it is in awaiting_approval status,
// updates the status to pending, and triggers the deploy workflow.
func (s *Service) AuthorizeDeployment(ctx context.Context, req *connect.Request[ctrlv1.AuthorizeDeploymentRequest]) (*connect.Response[ctrlv1.AuthorizeDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

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

	// Look up build settings and repo connection before changing status,
	// so a lookup failure doesn't leave the deployment stuck as pending.
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

	// Atomically transition from awaiting_approval → pending to prevent
	// concurrent authorization requests from triggering duplicate deploys.
	casResult, err := db.Query.CompareAndSwapDeploymentStatus(ctx, s.db.RW(), db.CompareAndSwapDeploymentStatusParams{
		ID:             deploymentID,
		ExpectedStatus: db.DeploymentsStatusAwaitingApproval,
		NewStatus:      db.DeploymentsStatusPending,
		UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment status: %w", err))
	}
	rowsAffected, err := casResult.RowsAffected()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check rows affected: %w", err))
	}
	if rowsAffected == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s is no longer awaiting approval (concurrent update)", deploymentID))
	}

	commitSHA := ""
	if deployment.GitCommitSha.Valid {
		commitSHA = deployment.GitCommitSha.String
	}

	var prNumber int64
	if deployment.PrNumber.Valid {
		prNumber = deployment.PrNumber.Int64
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
				PrNumber:       prNumber,
			},
		},
	}

	// Keyed by deployment_id — each deployment runs as its own isolated
	// workflow so multiple deployments can build in parallel.
	invocation, sendErr := s.deploymentClient(deploymentID).Deploy().Send(ctx, deployReq)
	if sendErr != nil {
		// Revert status back to awaiting_approval since the deploy failed.
		if _, revertErr := db.Query.CompareAndSwapDeploymentStatus(ctx, s.db.RW(), db.CompareAndSwapDeploymentStatusParams{
			ID:             deploymentID,
			ExpectedStatus: db.DeploymentsStatusPending,
			NewStatus:      db.DeploymentsStatusAwaitingApproval,
			UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		}); revertErr != nil {
			logger.Error("failed to revert deployment status after deploy failure",
				"deployment_id", deploymentID,
				"error", revertErr,
			)
		}
		logger.Error("failed to trigger deploy workflow after authorization",
			"deployment_id", deploymentID,
			"error", sendErr,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deploy workflow: %w", sendErr))
	}

	// Persist the invocation ID so the deployment can be cancelled later.
	invocationID := invocation.Id()
	if updateErr := db.Query.UpdateDeploymentInvocationID(ctx, s.db.RW(), db.UpdateDeploymentInvocationIDParams{
		ID:           deploymentID,
		InvocationID: sql.NullString{Valid: true, String: invocationID},
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); updateErr != nil {
		logger.Error("failed to persist invocation id",
			"deployment_id", deploymentID,
			"invocation_id", invocationID,
			"error", updateErr,
		)
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
