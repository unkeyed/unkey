package v1RatelimitRatelimit

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/btcsuite/btcutil/base58"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"google.golang.org/protobuf/proto"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/ratelimit.v1.RatelimitService/Ratelimit", func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		req := &openapi.V1RatelimitRatelimitRequestBody{}
		err := svc.OpenApiValidator.Body(c, req)
		if err != nil {
			return err
		}

		if req.Cost == nil {
			req.Cost = util.Pointer[int64](1)
		}

		var lease *ratelimitv1.LeaseRequest = nil
		if req.Lease != nil {
			lease = &ratelimitv1.LeaseRequest{
				Cost:    req.Lease.Cost,
				Timeout: req.Lease.Timeout,
			}
		}

		res, err := svc.Ratelimit.Ratelimit(ctx, &ratelimitv1.RatelimitRequest{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
			Cost:       *req.Cost,
			Lease:      lease,
		})
		if err != nil {
			return fault.Wrap(err, fmsg.With("failed to ratelimit"))
		}

		response := openapi.V1RatelimitRatelimitResponseBody{
			Limit:     res.Limit,
			Remaining: res.Remaining,
			Reset:     res.Reset_,
			Success:   res.Success,
			Current:   res.Current,
		}

		if res.Lease != nil {
			b, err := proto.Marshal(res.Lease)
			if err != nil {
				return fault.Wrap(err, fmsg.With("failed to marshal lease"))
			}
			response.Lease = base58.Encode(b)
		}

		return c.JSON(response)
	})
}
