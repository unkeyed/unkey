package version

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
	// Query version from database
	version, err := db.Query.FindVersionById(ctx, s.db.RO(), req.Msg.GetVersionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Convert database model to proto
	protoVersion := &ctrlv1.Version{
		Id:                   version.ID,
		WorkspaceId:          version.WorkspaceID,
		ProjectId:            version.ProjectID,
		EnvironmentId:        "", // No longer in schema
		Status:               convertDbStatusToProto(string(version.Status)),
		CreatedAt:            version.CreatedAt,
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

	if version.GitCommitSha.Valid {
		protoVersion.GitCommitSha = version.GitCommitSha.String
	}
	if version.GitBranch.Valid {
		protoVersion.GitBranch = version.GitBranch.String
	}
	if version.UpdatedAt.Valid {
		protoVersion.UpdatedAt = version.UpdatedAt.Int64
	}
	if version.RootfsImageID != "" {
		protoVersion.RootfsImageId = version.RootfsImageID
	}

	// Find the latest build for this version
	build, err := db.Query.FindLatestBuildByVersionId(ctx, s.db.RO(), version.ID)
	if err == nil {
		protoVersion.BuildId = build.ID
	}

	// Fetch version steps
	versionSteps, err := db.Query.FindVersionStepsByVersionId(ctx, s.db.RO(), version.ID)
	if err != nil {
		s.logger.Warn("failed to fetch version steps", "error", err, "version_id", version.ID)
		// Continue without steps rather than failing the entire request
	} else {
		protoSteps := make([]*ctrlv1.VersionStep, len(versionSteps))
		for i, step := range versionSteps {
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

	// Fetch routes (hostnames) for this version
	routes, err := db.Query.FindHostnameRoutesByVersionId(ctx, s.db.RO(), version.ID)
	if err != nil {
		s.logger.Warn("failed to fetch routes for version", "error", err, "version_id", version.ID)
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
