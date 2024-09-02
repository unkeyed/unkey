package v1RatelimitCommitLease

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/btcsuite/btcutil/base58"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"google.golang.org/protobuf/proto"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/ratelimit.v1.RatelimitService/CommitLease",
		func(c *fiber.Ctx) error {
			ctx := c.UserContext()

			req := &openapi.V1RatelimitCommitLeaseRequestBody{}
			err := svc.OpenApiValidator.Body(c, req)
			if err != nil {
				return err
			}

			b := base58.Decode(req.Lease)
			lease := &ratelimitv1.Lease{}
			err = proto.Unmarshal(b, lease)
			if err != nil {
				return fault.Wrap(err, fmsg.WithDesc("invalid_lease", "The lease is not valid."))
			}

			_, err = svc.Ratelimit.CommitLease(ctx, &ratelimitv1.CommitLeaseRequest{
				Lease: lease,
				Cost:  req.Cost,
			})
			if err != nil {
				return fault.Wrap(err, fmsg.With("failed to commit lease"))
			}

			return c.SendStatus(200)
		})
}
