package apis

import (
	"context"
)

type ApiService interface {
	CreateApi(context.Context, CreateApiRequest) (CreateApiResponse, error)
	RemoveApi(context.Context, RemoveApiRequest) (RemoveApiResponse, error)
}

type CreateApiRequest struct {
	Name        string
	WorkspaceId string
}

type CreateApiResponse struct {
	ApiId string
}

type RemoveApiRequest struct {
	ApiId string
}
type RemoveApiResponse struct{}
