package workspaces

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

type WorkspaceService interface {
	CreateWorkspace(context.Context, CreateWorkspaceRequest) (CreateWorkspaceResponse, error)
	ChangePlan(context.Context, ChangePlanRequest) (ChangePlanResponse, error)
}

type CreateWorkspaceRequest struct {
	Name     string
	Slug     string
	TenantId string
	Plan     entities.Plan
}

type CreateWorkspaceResponse struct {
	entities.Workspace
}

type ChangePlanRequest struct {
	WorkspaceId string
	Plan        entities.Plan
}
type ChangePlanResponse struct{}
