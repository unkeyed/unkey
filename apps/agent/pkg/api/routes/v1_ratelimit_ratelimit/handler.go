package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type V1RatelimitRatelimitRequest struct {
	Body struct {
		Identifier string `json:"identifier" required:"true" doc:"The identifier for the rate limit."`
		Limit      int64  `json:"limit" required:"true" doc:"The maximum number of requests allowed."`
		Duration   int64  `json:"duration" required:"true" doc:"The duration in milliseconds for the rate limit window."`
		Cost       int64  `json:"cost" required:"false" default:"1" doc:"The cost of the request."`
	}
}

type V1RatelimitRatelimitResponse struct {
	Body struct {
		Limit     int64 `json:"limit" doc:"The maximum number of requests allowed."`
		Remaining int64 `json:"remaining" doc:"The number of requests remaining in the current window."`
		Reset     int64 `json:"reset" doc:"The time in milliseconds when the rate limit will reset."`
		Success   bool  `json:"success" doc:"Whether the request passed the ratelimit. If false, the request must be blocked."`
		Current   int64 `json:"current" doc:"The current number of requests made in the current window."`
	}
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"ratelimit"},
		OperationID: "ratelimit.v1.ratelimit",
		Method:      "POST",
		Path:        "/ratelimit.v1.RatelimitService/Ratelimit",
		Middlewares: middlewares,
	}, func(ctx context.Context, req *V1RatelimitRatelimitRequest) (*V1RatelimitRatelimitResponse, error) {

		ctx, span := tracing.Start(ctx, tracing.NewSpanName("ratelimit", "Ratelimit"))
		defer span.End()

		res, err := svc.Ratelimit.Ratelimit(ctx, &ratelimitv1.RatelimitRequest{
			Identifier: req.Body.Identifier,
			Limit:      req.Body.Limit,
			Duration:   req.Body.Duration,
			Cost:       req.Body.Cost,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("unable to ratelimit", err)
		}

		response := V1RatelimitRatelimitResponse{}
		response.Body.Limit = res.Limit
		response.Body.Remaining = res.Remaining
		response.Body.Reset = res.Reset_
		response.Body.Success = res.Success
		response.Body.Current = res.Current

		return &response, nil
	})
}
