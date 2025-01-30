package ratelimit

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/Southclaws/fault"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"go.opentelemetry.io/otel/attribute"
)

func (s *service) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {
	ctx, span := tracing.Start(ctx, "ratelimit.Ratelimit")
	defer span.End()

	now := time.Now()
	if req.Time != nil {
		now = time.UnixMilli(req.GetTime())
	} else {
		req.Time = util.Pointer(now.UnixMilli())
	}

	ratelimitReq := ratelimitRequest{
		Time:       now,
		Name:       req.Name,
		Identifier: req.Identifier,
		Limit:      req.Limit,
		Duration:   time.Duration(req.Duration) * time.Millisecond,
		Cost:       req.Cost,
	}
	if req.Lease != nil {
		ratelimitReq.Lease = &lease{
			Cost:      req.Lease.Cost,
			ExpiresAt: now.Add(time.Duration(req.Lease.Timeout) * time.Millisecond),
		}
	}

	prevExists, currExists := s.CheckWindows(ctx, ratelimitReq)
	// If neither window existed before, we should do an origin ratelimit check
	// because we likely don't have enough data to make an accurate decision'
	if !prevExists && !currExists {

		originRes, err := s.ratelimitOrigin(ctx, req)
		// The control flow is a bit unusual here because we want to return early on
		// success, rather than on error
		if err == nil && originRes != nil {
			return originRes, nil
		}
		if err != nil {
			// We want to know about the error, but if there is one, we just fall back
			// to local state, so we don't return early
			s.logger.Warn().Err(err).Msg("failed to sync with origin, falling back to local state")
		}
	}

	taken := s.Take(ctx, ratelimitReq)

	// s.logger.Warn().Str("taken", fmt.Sprintf("%+v", taken)).Send()

	res := &ratelimitv1.RatelimitResponse{
		Current:   int64(taken.Current),
		Limit:     int64(taken.Limit),
		Remaining: int64(taken.Remaining),
		Reset_:    taken.Reset,
		Success:   taken.Pass,
	}

	if s.syncBuffer != nil {
		err := s.bufferSync(ctx, req, now, res.Success)
		if err != nil {
			s.logger.Err(err).Msg("failed to sync buffer")
		}
	}

	if req.Lease != nil {
		res.Lease = &ratelimitv1.Lease{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
		}
	}

	return res, nil

}

// ratelimitOrigin forwards the ratelimit request to the origin node and updates
// the local state to reflect the true state
func (s *service) ratelimitOrigin(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {
	ctx, span := tracing.Start(ctx, "ratelimit.RatelimitOrigin")
	defer span.End()

	forceSync.Inc()

	now := time.Now()
	if req.Time != nil {
		now = time.UnixMilli(req.GetTime())
	}

	key := bucketKey{req.Identifier, req.Limit, time.Duration(req.Duration) * time.Millisecond}

	client, peer, err := s.getPeerClient(ctx, key.toString())
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Err(err).Msg("failed to get peer client")
		return nil, err
	}
	if peer.Id == s.cluster.NodeId() {
		return nil, nil
	}
	s.logger.Info().Str("identifier", req.Identifier).Msg("no local state found, syncing with origin")

	connectReq := connect.NewRequest(&ratelimitv1.PushPullRequest{
		Request: req,
		Time:    now.UnixMilli(),
	})

	res, err := s.syncCircuitBreaker.Do(ctx, func(innerCtx context.Context) (*connect.Response[ratelimitv1.PushPullResponse], error) {
		innerCtx, cancel := context.WithTimeout(innerCtx, 10*time.Second)
		defer cancel()
		return client.PushPull(innerCtx, connectReq)
	})
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Msg("failed to call ratelimit")
		return nil, err
	}

	duration := time.Duration(req.Duration) * time.Millisecond
	err = s.SetCounter(ctx,
		setCounterRequest{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Counter:    res.Msg.Current.Counter,
			Sequence:   res.Msg.Current.Sequence,
			Duration:   duration,
			Time:       time.UnixMilli(req.GetTime()),
		},
		setCounterRequest{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Counter:    res.Msg.Previous.Counter,
			Sequence:   res.Msg.Previous.Sequence,
			Duration:   duration,
			Time:       time.UnixMilli(req.GetTime()),
		},
	)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Err(err).Msg("failed to set counter")
		return nil, err
	}
	return res.Msg.Response, nil
}

func (s *service) bufferSync(ctx context.Context, req *ratelimitv1.RatelimitRequest, now time.Time, localPassed bool) error {
	ctx, span := tracing.Start(ctx, "ratelimit.bufferSync")
	defer span.End()
	key := bucketKey{req.Identifier, req.Limit, time.Duration(req.Duration) * time.Millisecond}.toString()

	origin, err := s.cluster.FindNode(key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Str("key", key).Msg("unable to find responsible nodes")
		return fault.Wrap(err)
	}
	span.SetAttributes(attribute.Int("channelSize", len(s.syncBuffer)))
	s.logger.Debug().Str("origin", origin.Id).Int("size", len(s.syncBuffer)).Msg("syncing with origin")
	if origin.Id == s.cluster.NodeId() {
		// no need to sync with ourselves
		return nil
	}
	s.syncBuffer <- syncWithOriginRequest{
		req: &ratelimitv1.PushPullRequest{
			Passed:  localPassed,
			Time:    now.UnixMilli(),
			Request: req,
		},
		localPassed: localPassed,
	}
	return nil
}
