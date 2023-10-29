package workspaces

import (
	"context"

	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

type database interface {
	InsertWorkspace(ctx context.Context, workspace *workspacesv1.Workspace) error
	FindWorkspace(ctx context.Context, workspaceId string) (*workspacesv1.Workspace, bool, error)
	DeleteWorkspace(ctx context.Context, workspaceId string) error
	UpdateWorkspace(ctx context.Context, workspace *workspacesv1.Workspace) error
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
