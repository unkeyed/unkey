package deployment

import (
	"context"
	"database/sql"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetDeploymentRequest],
) (*connect.Response[ctrlv1.GetDeploymentResponse], error) {
	// Query deployment from database
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetDeploymentId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		s.logger.Error("failed to load deployment", "error", err, "deployment_id", req.Msg.GetDeploymentId())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployment"))
	}

	// Convert database model to proto
	protoDeployment := &ctrlv1.Deployment{
		Id:                   deployment.ID,
		WorkspaceId:          deployment.WorkspaceID,
		ProjectId:            deployment.ProjectID,
		EnvironmentId:        deployment.EnvironmentID,
		Status:               convertDbStatusToProto(deployment.Status),
		CreatedAt:            deployment.CreatedAt,
		GitCommitSha:         "",
		GitBranch:            "",
		ErrorMessage:         "",
		EnvironmentVariables: nil,
		Topology:             nil,
		UpdatedAt:            0,
		Hostnames:            nil,
		RootfsImageId:        "",
		BuildId:              "",
		Steps:                nil,
	}

	if deployment.GitCommitSha.Valid {
		protoDeployment.GitCommitSha = deployment.GitCommitSha.String
	}
	if deployment.GitBranch.Valid {
		protoDeployment.GitBranch = deployment.GitBranch.String
	}
	if deployment.GitCommitMessage.Valid {
		protoDeployment.GitCommitMessage = deployment.GitCommitMessage.String
	}

	// Email removed to avoid storing PII - TODO: implement GitHub API lookup
	if deployment.GitCommitAuthorHandle.Valid {
		protoDeployment.GitCommitAuthorHandle = deployment.GitCommitAuthorHandle.String
	}
	if deployment.GitCommitAuthorAvatarUrl.Valid {
		protoDeployment.GitCommitAuthorAvatarUrl = deployment.GitCommitAuthorAvatarUrl.String
	}
	if deployment.GitCommitTimestamp.Valid {
		protoDeployment.GitCommitTimestamp = deployment.GitCommitTimestamp.Int64
	}
	if deployment.UpdatedAt.Valid {
		protoDeployment.UpdatedAt = deployment.UpdatedAt.Int64
	}

	// Fetch deployment steps
	deploymentSteps, err := db.Query.FindDeploymentStepsByDeploymentId(ctx, s.db.RO(), deployment.ID)
	if err != nil {
		s.logger.Warn("failed to fetch deployment steps", "error", err, "deployment_id", deployment.ID)
		// Continue without steps rather than failing the entire request
	} else {
		protoSteps := make([]*ctrlv1.DeploymentStep, len(deploymentSteps))
		for i, step := range deploymentSteps {
			protoSteps[i] = &ctrlv1.DeploymentStep{
				Status:    string(step.Status),
				CreatedAt: step.CreatedAt,
				Message:   step.Message,
			}
		}
		protoDeployment.Steps = protoSteps
	}

	// Fetch routes (hostnames) for this deployment
	routes, err := db.Query.FindDomainsByDeploymentId(ctx, s.db.RO(), sql.NullString{Valid: true, String: req.Msg.GetDeploymentId()})
	if err != nil {
		s.logger.Warn("failed to fetch domains for deployment", "error", err, "deployment_id", deployment.ID)
		// Continue without hostnames rather than failing the entire request
	} else {
		hostnames := make([]string, len(routes))
		for i, route := range routes {
			hostnames[i] = route.Domain
		}
		protoDeployment.Hostnames = hostnames
	}

	res := connect.NewResponse(&ctrlv1.GetDeploymentResponse{
		Deployment: protoDeployment,
	})

	return res, nil
}

func convertDbStatusToProto(status db.DeploymentsStatus) ctrlv1.DeploymentStatus {
	switch status {
	case db.DeploymentsStatusPending:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	case db.DeploymentsStatusBuilding:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_BUILDING
	case db.DeploymentsStatusDeploying:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_DEPLOYING
	case db.DeploymentsStatusNetwork:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_NETWORK
	case db.DeploymentsStatusReady:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY
	case db.DeploymentsStatusFailed:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED
	default:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
	}
}
