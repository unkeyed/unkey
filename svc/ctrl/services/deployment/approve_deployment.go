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
)

// ApproveDeployment transitions a deployment from awaiting_approval to pending
// and triggers the deploy workflow. It records an approval audit entry.
func (s *Service) ApproveDeployment(ctx context.Context, req *connect.Request[ctrlv1.ApproveDeploymentRequest]) (*connect.Response[ctrlv1.ApproveDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	approvedBy := req.Msg.GetApprovedBy()

	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", deploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find deployment: %w", err))
	}

	if deployment.Status != db.DeploymentsStatusAwaitingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s is not awaiting approval (status: %s)", deploymentID, deployment.Status))
	}

	// Look up github repo connection for the git source
	repoConn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, s.db.RO(), deployment.AppID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("no GitHub connection found for app %s", deployment.AppID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find repo connection: %w", err))
	}

	// Look up build settings for dockerfile/context paths
	buildSettings, err := db.Query.FindAppBuildSettingByAppEnv(ctx, s.db.RO(), db.FindAppBuildSettingByAppEnvParams{
		AppID:         deployment.AppID,
		EnvironmentID: deployment.EnvironmentID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("no build settings found for app %s", deployment.AppID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find build settings: %w", err))
	}

	now := time.Now().UnixMilli()

	// Update status and record approval in a transaction
	err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.Query.UpdateDeploymentStatus(txCtx, tx, db.UpdateDeploymentStatusParams{
			Status:    db.DeploymentsStatusPending,
			UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
			ID:        deploymentID,
		}); txErr != nil {
			return txErr
		}

		return db.Query.InsertDeploymentApproval(txCtx, tx, db.InsertDeploymentApprovalParams{
			DeploymentID: deploymentID,
			ApprovedBy:   approvedBy,
			ApprovedAt:   now,
			SenderLogin:  "", // populated from the original push event if available
		})
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to approve deployment: %w", err))
	}

	// Trigger deploy workflow
	commitSHA := ""
	if deployment.GitCommitSha.Valid {
		commitSHA = deployment.GitCommitSha.String
	}

	_, err = s.deploymentClient(deployment.ProjectID).
		Deploy().
		Send(ctx, &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repoConn.InstallationID,
					Repository:     repoConn.RepositoryFullName,
					CommitSha:      commitSHA,
					ContextPath:    buildSettings.DockerContext,
					DockerfilePath: buildSettings.Dockerfile,
				},
			},
		})
	if err != nil {
		logger.Error("failed to trigger deploy workflow after approval",
			"deployment_id", deploymentID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deploy: %w", err))
	}

	logger.Info("deployment approved and deploy workflow triggered",
		"deployment_id", deploymentID,
		"approved_by", approvedBy,
	)

	return connect.NewResponse(&ctrlv1.ApproveDeploymentResponse{}), nil
}
