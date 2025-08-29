package deployment

import (
	"context"
	"database/sql"

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
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Convert database model to proto
	protoDeployment := &ctrlv1.Deployment{
		Id:                   deployment.ID,
		WorkspaceId:          deployment.WorkspaceID,
		ProjectId:            deployment.ProjectID,
		EnvironmentId:        deployment.EnvironmentID,
		Status:               convertDbStatusToProto(string(deployment.Status)),
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

func convertDbStatusToProto(status string) ctrlv1.DeploymentStatus {
	switch status {
	case "pending":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	case "building":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_BUILDING
	case "deploying":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_DEPLOYING
	case "active":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_ACTIVE
	case "failed":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED
	case "archived":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_ARCHIVED
	default:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
	}
}
