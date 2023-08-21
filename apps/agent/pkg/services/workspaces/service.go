package workspaces

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

type database interface {
	FindWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, bool, error)
	InsertWorkspace(ctx context.Context, newWorkspace entities.Workspace) error
	UpdateWorkspace(ctx context.Context, workspace entities.Workspace) error
}

type service struct {
	database database
}

type Config struct {
	Database database
}

type Middleware func(WorkspaceService) WorkspaceService

func New(config Config, mws ...Middleware) WorkspaceService {
	var svc WorkspaceService = &service{
		database: config.Database,
	}

	for _, mw := range mws {
		svc = mw(svc)
	}
	return svc
}
