package apis

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func (s *service) CreateApi(ctx context.Context, req *apisv1.CreateApiRequest) (*apisv1.CreateApiResponse, error) {

	keyAuth := &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: req.WorkspaceId,
	}

	err := s.database.InsertKeyAuth(ctx, keyAuth)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}

	api := &apisv1.Api{
		ApiId:       uid.Api(),
		Name:        req.Name,
		WorkspaceId: req.WorkspaceId,
		IpWhitelist: []string{},
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   util.Pointer(keyAuth.KeyAuthId),
	}

	err = s.database.InsertApi(ctx, api)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	return &apisv1.CreateApiResponse{ApiId: api.ApiId}, nil
}
