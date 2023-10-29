package workspaces

import (
	"context"

	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

type WorkspaceService interface {
	CreateWorkspace(context.Context, *workspacesv1.CreateWorkspaceRequest) (*workspacesv1.CreateWorkspaceResponse, error)
	FindWorkspace(context.Context, *workspacesv1.FindWorkspaceRequest) (*workspacesv1.FindWorkspaceResponse, error)
	RenameWorkspace(context.Context, *workspacesv1.RenameWorkspaceRequest) (*workspacesv1.RenameWorkspaceResponse, error)
	ChangePlan(context.Context, *workspacesv1.ChangePlanRequest) (*workspacesv1.ChangePlanResponse, error)
	DeleteWorkspace(context.Context, *workspacesv1.DeleteWorkspaceRequest) (*workspacesv1.DeleteWorkspaceResponse, error)
}
