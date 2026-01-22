package deployment

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
)

// CreateS3UploadURL generates a presigned S3 URL for uploading a build context
// archive. The URL is valid for 15 minutes. The build context path is generated
// using the project ID and a unique build ID, formatted as
// "{project_id}/{build_id}.tar.gz". Clients should upload a tar.gz archive
// containing the application source code to this URL, then pass the returned
// BuildContextPath to [CreateDeployment].
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
