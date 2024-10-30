package v1RatelimitMultiRatelimit

import (
	"net/http"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/ratelimit.v1.RatelimitService/MultiRatelimit", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req := &openapi.V1RatelimitMultiRatelimitRequestBody{}
		res := &openapi.V1RatelimitMultiRatelimitResponseBody{}

		errorResponse, valid := svc.OpenApiValidator.Body(r, req)
		if !valid {
			svc.Sender.Send(ctx, w, 400, errorResponse)
			return
		}
		ratelimits := make([]*ratelimitv1.RatelimitRequest, len(req.Ratelimits))
		for i, r := range req.Ratelimits {
			cost := int64(1)
			if r.Cost != nil {
				cost = *r.Cost
			}
			ratelimits[i] = &ratelimitv1.RatelimitRequest{
				Identifier: r.Identifier,
				Limit:      r.Limit,
				Duration:   r.Duration,
				Cost:       cost,
			}
		}
		svcRes, err := svc.Ratelimit.MultiRatelimit(ctx, &ratelimitv1.RatelimitMultiRequest{})
		if err != nil {
			errors.HandleError(ctx, err)
			return

		}
		res.Ratelimits = make([]openapi.SingleRatelimitResponse, len(res.Ratelimits))
		for i, r := range svcRes.Ratelimits {
			res.Ratelimits[i] = openapi.SingleRatelimitResponse{
				Current:   r.Current,
				Limit:     r.Limit,
				Remaining: r.Remaining,
				Reset:     r.Reset_,
				Success:   r.Success,
			}
		}

		svc.Sender.Send(ctx, w, 200, res)
	})
}
