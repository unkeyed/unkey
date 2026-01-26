package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// GetDeployment retrieves a deployment by ID including its current status,
// git metadata, and associated hostnames. Returns [connect.CodeNotFound] if the
// deployment does not exist. Hostnames are fetched separately from frontline
// routes; if that lookup fails, the response still succeeds but with an empty
// hostname list.
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
		GitCommitMessage:         "",
		GitCommitAuthorHandle:    "",
		GitCommitAuthorAvatarUrl: "",
		GitCommitTimestamp:       0,
		Id:                       deployment.ID,
		WorkspaceId:              deployment.WorkspaceID,
		ProjectId:                deployment.ProjectID,
		EnvironmentId:            deployment.EnvironmentID,
		Status:                   convertDbStatusToProto(deployment.Status),
		CreatedAt:                deployment.CreatedAt,
		GitCommitSha:             "",
		GitBranch:                "",
		ErrorMessage:             "",
		EnvironmentVariables:     nil,
		Topology:                 nil,
		UpdatedAt:                0,
		Hostnames:                nil,
		RootfsImageId:            "",
		BuildId:                  "",
		Steps:                    nil,
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

	// Fetch routes (fqdns) for this deployment
	routes, err := db.Query.FindFrontlineRoutesByDeploymentID(ctx, s.db.RO(), req.Msg.GetDeploymentId())
	if err != nil {
		s.logger.Warn("failed to fetch frontline routes for deployment", "error", err, "deployment_id", deployment.ID)
		// Continue without fqdns rather than failing the entire request
	} else {
		fqdns := make([]string, len(routes))
		for i, route := range routes {
			fqdns[i] = route.FullyQualifiedDomainName
		}
		protoDeployment.Hostnames = fqdns
	}

	res := connect.NewResponse(&ctrlv1.GetDeploymentResponse{
		Deployment: protoDeployment,
	})

	return res, nil
}

// convertDbStatusToProto maps database deployment status to the proto enum.
// Returns DEPLOYMENT_STATUS_UNSPECIFIED for unknown status values.
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
