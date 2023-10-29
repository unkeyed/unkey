package apis

import (
	"context"
	"fmt"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

func (s *service) FindApi(ctx context.Context, req *apisv1.FindApiRequest) (*apisv1.FindApiResponse, error) {

	api, found, err := s.database.FindApi(ctx, req.ApiId)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	if !found {
		return nil, errors.New(errors.ErrNotFound, fmt.Errorf("api %s does not exist", req.ApiId))
	}
	return &apisv1.FindApiResponse{Api: api}, nil
}
