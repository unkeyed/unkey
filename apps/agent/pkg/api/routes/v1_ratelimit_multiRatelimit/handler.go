package v1RatelimitMultiRatelimit

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type v1RatelimitMultiRatelimitRequest struct {
	Body struct {
		Ratelimits []struct {
			Identifier string `json:"identifier" required:"true" doc:"The identifier for the rate limit."`
			Limit      int64  `json:"limit" required:"true" doc:"The maximum number of requests allowed."`
			Duration   int64  `json:"duration" required:"true" doc:"The duration in milliseconds for the rate limit window."`
			Cost       int64  `json:"cost" required:"false" default:"1" doc:"The cost of the request."`
		} `json:"ratelimits" required:"true" doc:"The rate limits to check."`
	}
}

// singleRatelimitResponse is the response for a single ratelimit request.
// This struct is used for the response body of the ratelimit endpoint and the multiRatelimit endpoint.
type singleRatelimitResponse struct {
	Limit     int64 `json:"limit" doc:"The maximum number of requests allowed."`
	Remaining int64 `json:"remaining" doc:"The number of requests remaining in the current window."`
	Reset     int64 `json:"reset" doc:"The time in milliseconds when the rate limit will reset."`
	Success   bool  `json:"success" doc:"Whether the request passed the ratelimit. If false, the request must be blocked."`
	Current   int64 `json:"current" doc:"The current number of requests made in the current window."`
}
type v1RatelimitRatelimitResponse struct {
	Body singleRatelimitResponse
}

type v1RatelimitMultiRatelimitResponse struct {
	Body struct {
		Ratelimits []singleRatelimitResponse `json:"ratelimits" doc:"The rate limits that were checked."`
	}
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"ratelimit"},
		OperationID: "ratelimit.v1.multiRatelimit",
		Method:      "POST",
		Path:        "/ratelimit.v1.RatelimitService/MultiRatelimit",
		Middlewares: middlewares,
	}, func(ctx context.Context, req *v1RatelimitMultiRatelimitRequest) (*v1RatelimitMultiRatelimitResponse, error) {

		ctx, span := tracing.Start(ctx, tracing.NewSpanName("ratelimit", "Ratelimit"))
		defer span.End()

		// Default cost is 1 if not provided

		ratelimits := make([]*ratelimitv1.RatelimitRequest, len(req.Body.Ratelimits))
		for i, r := range req.Body.Ratelimits {

			ratelimits[i] = &ratelimitv1.RatelimitRequest{
				Identifier: r.Identifier,
				Limit:      r.Limit,
				Duration:   r.Duration,
				Cost:       r.Cost,
			}
		}

		res, err := svc.Ratelimit.MultiRatelimit(ctx, &ratelimitv1.RatelimitMultiRequest{Ratelimits: ratelimits})
		if err != nil {
			return nil, huma.Error500InternalServerError("unable to ratelimit", err)
		}

		response := v1RatelimitMultiRatelimitResponse{}
		response.Body.Ratelimits = make([]singleRatelimitResponse, len(res.Ratelimits))
		for i, r := range res.Ratelimits {
			response.Body.Ratelimits[i] = singleRatelimitResponse{
				Current:   r.Current,
				Limit:     r.Limit,
				Remaining: r.Remaining,
				Reset:     r.Reset_,
				Success:   r.Success,
			}
		}

		return &response, nil
	})
}
