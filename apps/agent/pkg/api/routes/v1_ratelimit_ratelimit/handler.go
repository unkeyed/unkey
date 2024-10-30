package v1RatelimitRatelimit

import (
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"google.golang.org/protobuf/proto"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/ratelimit.v1.RatelimitService/Ratelimit", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		req := &openapi.V1RatelimitRatelimitRequestBody{}
		errorResponse, valid := svc.OpenApiValidator.Body(r, req)
		if !valid {
			svc.Sender.Send(ctx, w, 400, errorResponse)
			return
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
			errors.HandleError(ctx, err)
			return
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
				errors.HandleError(ctx, err)
				return
			}
			response.Lease = base58.Encode(b)
		}

		svc.Sender.Send(ctx, w, 200, response)
	})
}
