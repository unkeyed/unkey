package ratelimit

import (
	"context"
	"fmt"

	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
)

func (s *service) CommitLease(ctx context.Context, req *ratelimitv1.CommitLeaseRequest) (*ratelimitv1.CommitLeaseResponse, error) {
	// ctx, span := tracing.Start(ctx, "svc.ratelimit.CommitLease")
	// defer span.End()

	// key := bucketKey{req.Lease.Identifier, req.Lease.Limit, time.Duration(req.Lease.Duration) * time.Millisecond}

	// client, origin, err := s.getPeerClient(ctx, key.toString())
	// if err != nil {
	// 	tracing.RecordError(span, err)
	// 	s.logger.Warn().Err(err).Str("key", key.toString()).Msg("unable to find responsible nodes")
	// 	return nil, nil
	// }

	// // If we're the origin, we can commit the lease locally and return
	// if origin.Id == s.cluster.NodeId() {
	// 	s.commitLease(ctx, commitLeaseRequest{
	// 		Identifier: req.Lease.Identifier,
	// 		LeaseId:    "TODO",
	// 		Tokens:     req.Cost,
	// 	})

	// 	return &ratelimitv1.CommitLeaseResponse{}, nil
	// }

	// // Else we need to forward the request to the responsible node

	// connectReq := connect.NewRequest(req)

	// connectReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", s.cluster.AuthToken()))

	// res, err := client.CommitLease(ctx, connectReq)
	// if err != nil {
	// 	tracing.RecordError(span, err)
	// 	s.logger.Err(err).Msg("failed to commit lease")
	// 	return nil, fault.Wrap(err)
	// }
	// return res.Msg, nil
	return nil, fmt.Errorf("TODO: implement me")
}
