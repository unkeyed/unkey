package deployment

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetVersion(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetVersionRequest],
) (*connect.Response[ctrlv1.GetVersionResponse], error) {
	// Query deployment from database
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetVersionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Convert database model to proto
	protoVersion := &ctrlv1.Version{
		Id:                       deployment.ID,
		WorkspaceId:              deployment.WorkspaceID,
		ProjectId:                deployment.ProjectID,
		EnvironmentId:            string(deployment.Environment),
		Status:                   convertDbStatusToProto(string(deployment.Status)),
		CreatedAt:                deployment.CreatedAt,
		GitCommitSha:             "",
		GitBranch:                "",
		GitCommitMessage:         "",
		GitCommitAuthorName:      "",
		GitCommitAuthorEmail:     "",
		GitCommitAuthorUsername:  "",
		GitCommitAuthorAvatarUrl: "",
		GitCommitTimestamp:       0,
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
		protoVersion.GitCommitSha = deployment.GitCommitSha.String
	}
	if deployment.GitBranch.Valid {
		protoVersion.GitBranch = deployment.GitBranch.String
	}
	if deployment.GitCommitMessage.Valid {
		protoVersion.GitCommitMessage = deployment.GitCommitMessage.String
	}
	if deployment.GitCommitAuthorName.Valid {
		protoVersion.GitCommitAuthorName = deployment.GitCommitAuthorName.String
	}
	if deployment.GitCommitAuthorEmail.Valid {
		protoVersion.GitCommitAuthorEmail = deployment.GitCommitAuthorEmail.String
	}
	if deployment.GitCommitAuthorUsername.Valid {
		protoVersion.GitCommitAuthorUsername = deployment.GitCommitAuthorUsername.String
	}
	if deployment.GitCommitAuthorAvatarUrl.Valid {
		protoVersion.GitCommitAuthorAvatarUrl = deployment.GitCommitAuthorAvatarUrl.String
	}
	if deployment.GitCommitTimestamp.Valid {
		protoVersion.GitCommitTimestamp = deployment.GitCommitTimestamp.Int64
	}
	if deployment.UpdatedAt.Valid {
		protoVersion.UpdatedAt = deployment.UpdatedAt.Int64
	}
	if deployment.RootfsImageID != "" {
		protoVersion.RootfsImageId = deployment.RootfsImageID
	}

	// Find the latest build for this deployment
	build, err := db.Query.FindLatestBuildByDeploymentId(ctx, s.db.RO(), deployment.ID)
	if err == nil {
		protoVersion.BuildId = build.ID
	}

	// Fetch deployment steps
	deploymentSteps, err := db.Query.FindDeploymentStepsByDeploymentId(ctx, s.db.RO(), deployment.ID)
	if err != nil {
		s.logger.Warn("failed to fetch deployment steps", "error", err, "deployment_id", deployment.ID)
		// Continue without steps rather than failing the entire request
	} else {
		protoSteps := make([]*ctrlv1.VersionStep, len(deploymentSteps))
		for i, step := range deploymentSteps {
			protoSteps[i] = &ctrlv1.VersionStep{
				Status:       string(step.Status),
				CreatedAt:    step.CreatedAt,
				Message:      "",
				ErrorMessage: "",
			}
			if step.Message.Valid {
				protoSteps[i].Message = step.Message.String
			}
			if step.ErrorMessage.Valid {
				protoSteps[i].ErrorMessage = step.ErrorMessage.String
			}
		}
		protoVersion.Steps = protoSteps
	}

	// Fetch routes (hostnames) for this deployment
	routes, err := db.Query.FindHostnameRoutesByDeploymentId(ctx, s.db.RO(), deployment.ID)
	if err != nil {
		s.logger.Warn("failed to fetch routes for deployment", "error", err, "deployment_id", deployment.ID)
		// Continue without hostnames rather than failing the entire request
	} else {
		hostnames := make([]string, len(routes))
		for i, route := range routes {
			hostnames[i] = route.Hostname
		}
		protoVersion.Hostnames = hostnames
	}

	res := connect.NewResponse(&ctrlv1.GetVersionResponse{
		Version: protoVersion,
	})

	return res, nil
}

func convertDbStatusToProto(status string) ctrlv1.VersionStatus {
	switch status {
	case "pending":
		return ctrlv1.VersionStatus_VERSION_STATUS_PENDING
	case "building":
		return ctrlv1.VersionStatus_VERSION_STATUS_BUILDING
	case "deploying":
		return ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING
	case "active":
		return ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE
	case "failed":
		return ctrlv1.VersionStatus_VERSION_STATUS_FAILED
	case "archived":
		return ctrlv1.VersionStatus_VERSION_STATUS_ARCHIVED
	default:
		return ctrlv1.VersionStatus_VERSION_STATUS_UNSPECIFIED
	}
}
