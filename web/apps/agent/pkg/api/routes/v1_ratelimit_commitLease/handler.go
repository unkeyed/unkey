package v1RatelimitCommitLease

import (
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/btcsuite/btcutil/base58"
	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/errors"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/svc/agent/pkg/openapi"
	"google.golang.org/protobuf/proto"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("POST", "/ratelimit.v1.RatelimitService/CommitLease",
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			req := &openapi.V1RatelimitCommitLeaseRequestBody{}
			errorResponse, valid := svc.OpenApiValidator.Body(r, req)
			if !valid {
				svc.Sender.Send(ctx, w, 400, errorResponse)
				return
			}

			b := base58.Decode(req.Lease)
			lease := &ratelimitv1.Lease{}
			err := proto.Unmarshal(b, lease)
			if err != nil {
				errors.HandleValidationError(ctx, fault.Wrap(err, fmsg.WithDesc("invalid_lease", "The lease is not valid.")))
				return
			}

			_, err = svc.Ratelimit.CommitLease(ctx, &ratelimitv1.CommitLeaseRequest{
				Lease: lease,
				Cost:  req.Cost,
			})
			if err != nil {
				errors.HandleError(ctx, fault.Wrap(err, fmsg.With("failed to commit lease")))
				return

			}

			svc.Sender.Send(ctx, w, 204, nil)
		})
}
