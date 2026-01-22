package deployment

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
)

func (s *Service) CreateS3UploadURL(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateS3UploadURLRequest],
) (*connect.Response[ctrlv1.CreateS3UploadURLResponse], error) {

	buildContextPath := fmt.Sprintf("%s/%s.tar.gz", req.Msg.GetUnkeyProjectId(), uid.New("build"))

	url, err := s.buildStorage.GenerateUploadURL(ctx, buildContextPath, 15*time.Minute)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ctrlv1.CreateS3UploadURLResponse{
		UploadUrl:        url,
		BuildContextPath: buildContextPath,
	}), nil
}
