package apis

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
)

type database interface {
	FindApi(ctx context.Context, apiId string) (*apisv1.Api, bool, error)
	InsertApi(ctx context.Context, newApi *apisv1.Api) error
	DeleteApi(ctx context.Context, apiId string) error
	InsertKeyAuth(ctx context.Context, keyAuth *authenticationv1.KeyAuth) error
}

type service struct {
	database database
}

type Config struct {
	Database database
}

type Middleware func(ApiService) ApiService

func New(config Config, mws ...Middleware) ApiService {
	var svc ApiService = &service{
		database: config.Database,
	}

	for _, mw := range mws {
		svc = mw(svc)
	}
	return svc
}
