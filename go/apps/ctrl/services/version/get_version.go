package version

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		EnvironmentId:        version.EnvironmentID,
		Status:               convertDbStatusToProto(string(version.Status)),
		CreatedAt:            timestamppb.New(time.UnixMilli(version.CreatedAtM)),
		GitCommitSha:         "",
		GitBranch:            "",
		ErrorMessage:         "",
		EnvironmentVariables: nil,
		Topology:             nil,
		UpdatedAt:            nil,
		Hostnames:            nil,
		RootfsImageId:        "",
		BuildId:              "",
	}

	if version.GitCommitSha.Valid {
		protoVersion.GitCommitSha = version.GitCommitSha.String
	}
	if version.GitBranch.Valid {
		protoVersion.GitBranch = version.GitBranch.String
	}
	if version.UpdatedAtM.Valid {
		protoVersion.UpdatedAt = timestamppb.New(time.UnixMilli(version.UpdatedAtM.Int64))
	}
	if version.RootfsImageID != "" {
		protoVersion.RootfsImageId = version.RootfsImageID
	}

	// Find the latest build for this version
	build, err := db.Query.FindLatestBuildByVersionId(ctx, s.db.RO(), version.ID)
	if err == nil {
		protoVersion.BuildId = build.ID
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
