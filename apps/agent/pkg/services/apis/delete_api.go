package apis

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

func (s *service) DeleteApi(ctx context.Context, req *apisv1.DeleteApiRequest) (*apisv1.DeleteApiResponse, error) {

	err := s.database.DeleteApi(ctx, req.ApiId)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	return &apisv1.DeleteApiResponse{}, nil
}
