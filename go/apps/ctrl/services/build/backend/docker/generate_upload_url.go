package docker

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (s *Docker) GenerateUploadURL(
	ctx context.Context,
	req *connect.Request[ctrlv1.GenerateUploadURLRequest],
) (*connect.Response[ctrlv1.GenerateUploadURLResponse], error) {
	if req.Msg.UnkeyProjectId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unkeyProjectID is required"))
	}

	// Generate unique S3 key for this build context
	contextKey := fmt.Sprintf("%s/%d.tar.gz",
		req.Msg.UnkeyProjectId,
		time.Now().UnixNano())

	// Generate presigned URL (15 minutes expiration)
	uploadURL, err := s.storage.PutPresignedURL(ctx, contextKey, 15*time.Minute)
	if err != nil {
		s.logger.Error("Failed to generate presigned URL", "error", err, "context_key", contextKey)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to generate presigned URL: %w", err))
	}

	s.logger.Info("Generated upload URL", "context_key", contextKey, "unkey_project_id", req.Msg.UnkeyProjectId)

	return connect.NewResponse(&ctrlv1.GenerateUploadURLResponse{
		UploadUrl:  uploadURL,
		ContextKey: contextKey,
		ExpiresIn:  900, // 15 minutes
	}), nil
}
