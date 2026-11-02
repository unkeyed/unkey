package keyleak

import (
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

func (s *Service) NotifyBatch(ctx restate.Context, req *hydrav1.KeyLeakBatchRequest) (*hydrav1.KeyLeakResponse, error) {
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return s.disableBatch(rc, req)
	}, restate.WithName("disable leaked keys and record audits"),
		restate.WithInitialRetryInterval(time.Second), restate.WithMaxRetryInterval(time.Minute)); err != nil {
		return nil, err
	}

	client := hydrav1.NewKeyLeakServiceClient(ctx)
	for _, key := range req.GetKeys() {
		client.Notify().Send(key.GetNotification(), restate.WithIdempotencyKey(req.GetReportId()+":"+key.GetHash()))
	}

	return &hydrav1.KeyLeakResponse{}, nil
}
