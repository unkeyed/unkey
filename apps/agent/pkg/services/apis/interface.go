package apis

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
)

type ApiService interface {
	CreateApi(context.Context, *apisv1.CreateApiRequest) (*apisv1.CreateApiResponse, error)
	FindApi(context.Context, *apisv1.FindApiRequest) (*apisv1.FindApiResponse, error)
	DeleteApi(context.Context, *apisv1.DeleteApiRequest) (*apisv1.DeleteApiResponse, error)
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

type FindApiRequest struct {
	ApiId string
}
type FindApiResponse struct {
	Api *apisv1.Api
}
