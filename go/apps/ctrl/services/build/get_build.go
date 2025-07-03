package build

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) GetBuild(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetBuildRequest],
) (*connect.Response[ctrlv1.GetBuildResponse], error) {
	// Query build from database
	build, err := db.Query.FindBuildById(ctx, s.db.RO(), req.Msg.GetBuildId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Convert database model to proto
	protoBuild := &ctrlv1.Build{
		Id:          build.ID,
		WorkspaceId: build.WorkspaceID,
		ProjectId:   build.ProjectID,
		VersionId:   build.VersionID,
		Status:      convertDbBuildStatusToProto(string(build.Status)),
		CreatedAt:   timestamppb.New(time.UnixMilli(build.CreatedAtM)),
	}

	if build.UpdatedAtM.Valid {
		protoBuild.UpdatedAt = timestamppb.New(time.UnixMilli(build.UpdatedAtM.Int64))
	}
	if build.StartedAt.Valid {
		protoBuild.StartedAt = timestamppb.New(time.UnixMilli(build.StartedAt.Int64))
	}
	if build.CompletedAt.Valid {
		protoBuild.CompletedAt = timestamppb.New(time.UnixMilli(build.CompletedAt.Int64))
	}
	if build.ErrorMessage.Valid {
		protoBuild.ErrorMessage = build.ErrorMessage.String
	}
	if build.RootfsImageID.Valid {
		protoBuild.RootfsImageId = build.RootfsImageID.String
	}

	res := connect.NewResponse(&ctrlv1.GetBuildResponse{
		Build: protoBuild,
	})

	return res, nil
}

func convertDbBuildStatusToProto(status string) ctrlv1.BuildStatus {
	switch status {
	case "pending":
		return ctrlv1.BuildStatus_BUILD_STATUS_PENDING
	case "running":
		return ctrlv1.BuildStatus_BUILD_STATUS_RUNNING
	case "succeeded":
		return ctrlv1.BuildStatus_BUILD_STATUS_SUCCEEDED
	case "failed":
		return ctrlv1.BuildStatus_BUILD_STATUS_FAILED
	case "cancelled":
		return ctrlv1.BuildStatus_BUILD_STATUS_CANCELLED
	default:
		return ctrlv1.BuildStatus_BUILD_STATUS_UNSPECIFIED
	}
}
