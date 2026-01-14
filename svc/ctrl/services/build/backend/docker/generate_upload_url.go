package docker

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

func (s *Docker) GenerateUploadURL(
	ctx context.Context,
	req *connect.Request[ctrlv1.GenerateUploadURLRequest],
) (*connect.Response[ctrlv1.GenerateUploadURLResponse], error) {
	unkeyProjectID := req.Msg.GetUnkeyProjectId()
	if unkeyProjectID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unkeyProjectID is required"))
	}

	// This ensures the project exists. Without this check, callers could provide
	// arbitrary projectIds and generate unlimited upload URLs.
	_, err := db.Query.FindProjectById(ctx, s.db.RO(), unkeyProjectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("project not found: %s", unkeyProjectID))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Generate unique S3 key for this build context
	buildContextPath := fmt.Sprintf("%s/%d.tar.gz",
		unkeyProjectID,
		time.Now().UnixNano())

	// Generate presigned URL (15 minutes expiration)
	uploadURL, err := s.storage.GenerateUploadURL(ctx, buildContextPath, 15*time.Minute)
	if err != nil {
		s.logger.Error("Failed to generate presigned URL", "error", err, "context_key", buildContextPath)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to generate presigned URL: %w", err))
	}

	s.logger.Info("Generated upload URL", "context_key", buildContextPath, "unkey_project_id", unkeyProjectID)

	return connect.NewResponse(&ctrlv1.GenerateUploadURLResponse{
		UploadUrl:        uploadURL,
		BuildContextPath: buildContextPath,
		ExpiresIn:        900, // 15 minutes
	}), nil
}
