package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

type v1LivenessRequest struct {
	// Empty
}
type v1LivenessResponse struct {
	Body struct {
		Message string `json:"message" example:"OK" doc:"Whether we're alive or not"`
	}
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"liveness"},
		OperationID: "liveness",
		Method:      "GET",
		Path:        "/v1/liveness",
		Summary:     "Liveness check",
		Description: "This endpoint checks if the service is alive.",
		Errors:      []int{500},
	}, func(ctx context.Context, req *v1LivenessRequest) (*v1LivenessResponse, error) {
		res := &v1LivenessResponse{}
		res.Body.Message = "OK"
		svc.Logger.Info().Interface("response", res).Msg("incoming liveness check")
		return res, nil

	})

}
