package handler

import (
	"context"

	"github.com/btcsuite/btcutil/base58"
	"github.com/danielgtaylor/huma/v2"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"google.golang.org/protobuf/proto"
)

type V1RatelimitCommitLeaseRequest struct {
	Body struct {
		Lease string `json:"lease" required:"true" doc:"The lease you received from the ratelimit response."`
		Cost  int64  `json:"cost" required:"true" doc:"The actual cost of the request."`
	} `required:"true" contentType:"application/json"`
}

type V1RatelimitCommitLeaseResponse struct {
	// empty, we'll just ack this
}

func Register(api huma.API, svc routes.Services, middlewares ...func(ctx huma.Context, next func(huma.Context))) {
	huma.Register(api, huma.Operation{
		Tags:        []string{"ratelimit"},
		OperationID: "v1.ratelimit.commitLease",
		Method:      "POST",
		Path:        "/v1/ratelimit.commitLease",
		Middlewares: middlewares,
	}, func(ctx context.Context, req *V1RatelimitCommitLeaseRequest) (*V1RatelimitCommitLeaseResponse, error) {

		ctx, span := tracing.Start(ctx, tracing.NewSpanName("ratelimit", "commitLease"))
		defer span.End()

		b := base58.Decode(req.Body.Lease)
		lease := &ratelimitv1.Lease{}
		err := proto.Unmarshal(b, lease)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid lease", err)
		}

		_, err = svc.Ratelimit.CommitLease(ctx, &ratelimitv1.CommitLeaseRequest{
			Lease: lease,
			Cost:  req.Body.Cost,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("unable to ratelimit", err)
		}

		response := V1RatelimitCommitLeaseResponse{}

		return &response, nil
	})
}
