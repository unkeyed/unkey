package v1RatelimitMultiRatelimit

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/ratelimit.v1.RatelimitService/MultiRatelimit", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		req := &openapi.V1RatelimitMultiRatelimitRequestBody{}
		err := svc.OpenApiValidator.Body(c, req)
		if err != nil {
			return err
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
		res, err := svc.Ratelimit.MultiRatelimit(ctx, &ratelimitv1.RatelimitMultiRequest{})
		if err != nil {
			return fault.Wrap(err, fmsg.With("failed to ratelimit"))
		}

		resLimits := make([]openapi.SingleRatelimitResponse, len(res.Ratelimits))
		for i, r := range res.Ratelimits {
			resLimits[i] = openapi.SingleRatelimitResponse{
				Current:   r.Current,
				Limit:     r.Limit,
				Remaining: r.Remaining,
				Reset:     r.Reset_,
				Success:   r.Success,
			}
		}

		return c.JSON(openapi.V1RatelimitMultiRatelimitResponseBody{Ratelimits: resLimits})
	})
}
