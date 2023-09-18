package apis

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func (s *service) CreateApi(ctx context.Context, req CreateApiRequest) (CreateApiResponse, error) {

	ws, found, err := s.database.FindWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return CreateApiResponse{}, fmt.Errorf("unable to find workspace %s: %w", req.WorkspaceId, err)
	}
	if !found {
		return CreateApiResponse{}, fmt.Errorf("workspace not found: %s", req.WorkspaceId)
	}

	keyAuth := entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: ws.Id,
	}

	err = s.database.InsertKeyAuth(ctx, keyAuth)
	if err != nil {
		return CreateApiResponse{}, fmt.Errorf("unable to create key auth: %w", err)
	}

	api := entities.Api{
		Id:          uid.Api(),
		Name:        req.Name,
		WorkspaceId: ws.Id,
		IpWhitelist: []string{},
		AuthType:    entities.AuthTypeKey,
		KeyAuthId:   keyAuth.Id,
	}

	err = s.database.InsertApi(ctx, api)
	if err != nil {
		return CreateApiResponse{}, fmt.Errorf("unable to create api: %w", err)
	}
	return CreateApiResponse{ApiId: api.Id}, nil
}
