package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// RejectDeployment transitions a deployment from awaiting_approval to rejected
// and updates the GitHub deployment status to failure.
func (s *Service) RejectDeployment(ctx context.Context, req *connect.Request[ctrlv1.RejectDeploymentRequest]) (*connect.Response[ctrlv1.RejectDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()

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

	now := time.Now().UnixMilli()
	err = db.Query.UpdateDeploymentStatus(ctx, s.db.RW(), db.UpdateDeploymentStatusParams{
		Status:    db.DeploymentsStatusRejected,
		UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
		ID:        deploymentID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reject deployment: %w", err))
	}

	// Update GitHub deployment status to failure if we have a GitHub deployment ID.
	if deployment.GithubDeploymentID.Valid {
		repoConn, repoErr := db.Query.FindGithubRepoConnectionByAppId(ctx, s.db.RO(), deployment.AppID)
		if repoErr == nil {
			if ghErr := s.github.CreateDeploymentStatus(
				repoConn.InstallationID,
				repoConn.RepositoryFullName,
				deployment.GithubDeploymentID.Int64,
				"failure",
				"",
				"",
				"Deployment rejected",
			); ghErr != nil {
				logger.Error("failed to update GitHub deployment status on rejection",
					"deployment_id", deploymentID,
					"error", ghErr,
				)
			}
		}
	}

	logger.Info("deployment rejected",
		"deployment_id", deploymentID,
	)

	return connect.NewResponse(&ctrlv1.RejectDeploymentResponse{}), nil
}
