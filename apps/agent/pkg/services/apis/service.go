package apis

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

type database interface {
	FindWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, bool, error)
	InsertApi(ctx context.Context, newWorkspace entities.Api) error
	DeleteApi(ctx context.Context, apiId string) error
	InsertKeyAuth(ctx context.Context, workspace entities.KeyAuth) error
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
