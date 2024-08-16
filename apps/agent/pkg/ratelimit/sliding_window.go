package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"go.opentelemetry.io/otel/attribute"
)

func WindowSequence(t time.Time, d time.Duration) int64 {
	return time.Now().UnixMilli() / d.Milliseconds()
}

type window struct {
	sync.RWMutex

	// the window sequence calculated by taking the current timestamp in
	// milliseconds and dividing by the duration
	sequence int64

	// when this window starts
	start time.Time
	// when this window ends
	end time.Time

	counter int64

	// leaseId -> lease
	leases map[string]lease
}

type identifierBucket struct {
	sync.RWMutex
	identifier string
	duration   time.Duration
	windows    map[int64]*window
}

type slidingWindow struct {
	identifiersLock     sync.RWMutex
	identifiers         map[string]*identifierBucket
	leaseIdToKeyMapLock sync.RWMutex
	// Store a reference leaseId -> window key
	leaseIdToKeyMap map[string]string
	logger          logging.Logger
}

func NewSlidingWindow(logger logging.Logger) *slidingWindow {

	r := &slidingWindow{
		identifiersLock:     sync.RWMutex{},
		identifiers:         make(map[string]*identifierBucket),
		leaseIdToKeyMapLock: sync.RWMutex{},
		leaseIdToKeyMap:     make(map[string]string),
		logger:              logger,
	}

	repeat.Every(time.Minute, r.removeExpiredIdentifiers)
	return r

}

func (r *slidingWindow) removeExpiredIdentifiers() {
	r.identifiersLock.Lock()
	defer r.identifiersLock.Unlock()

	activeRatelimits.Set(float64(len(r.identifiers)))
	now := time.Now()
	for id, identifier := range r.identifiers {
		identifier.Lock()
		defer identifier.Unlock()
		for seq, w := range identifier.windows {
			w.Lock()
			defer w.Unlock()
			if now.After(w.end.Add(identifier.duration)) {
				delete(identifier.windows, seq)
			}
		}
		if len(identifier.windows) == 0 {
			delete(r.identifiers, id)
		}
	}
}

// Has returns true if there is already a record for the given identifier in the current window
func (r *slidingWindow) Has(ctx context.Context, identifier string, duration time.Duration) bool {
	ctx, span := tracing.Start(ctx, "slidingWindow.Has")
	defer span.End()
	key := BuildKey(identifier, duration)

	r.identifiersLock.RLock()
	_, ok := r.identifiers[key]
	r.identifiersLock.RUnlock()
	return ok
}

func (r *slidingWindow) Take(ctx context.Context, req RatelimitRequest) RatelimitResponse {
	ctx, span := tracing.Start(ctx, "slidingWindow.Take")
	defer span.End()

	key := BuildKey(req.Identifier, req.Duration)
	span.SetAttributes(attribute.String("key", key))

	r.identifiersLock.RLock(ctx)
	id, ok := r.identifiers[key]
	r.identifiersLock.RUnlock(ctx)
	span.SetAttributes(attribute.Bool("identifierExisted", ok))
	if !ok {
		reset := time.UnixMilli((time.Now().UnixMilli()/req.Duration.Milliseconds() + 1) * req.Duration.Milliseconds())
		id = newIdentifierWindow(key, reset)

		r.identifiersLock.Lock(ctx)
		r.identifiers[key] = id
		r.identifiersLock.Unlock(ctx)
	}
	id.Lock()
	defer id.Unlock()

	// Calculate the current count including all leases
	currentWithLeases := id.current
	if req.Lease != nil {
		currentWithLeases += req.Lease.Cost
	}
	for _, lease := range id.leases {
		if lease.expiresAt.Before(time.Now()) {
			currentWithLeases += lease.cost
		}
	}

	// Evaluate if the request should pass or not

	if currentWithLeases+req.Cost > req.Limit {
		ratelimitsCount.WithLabelValues("false").Inc()
		return RatelimitResponse{Pass: false, Remaining: req.Limit - id.current, Reset: id.reset.UnixMilli(), Limit: req.Limit, Current: id.current}
	}

	if req.Lease != nil {
		leaseId := uid.New("lease")
		id.leases[leaseId] = lease{
			id:        leaseId,
			cost:      req.Lease.Cost,
			expiresAt: req.Lease.ExpiresAt,
		}
		r.leaseIdToKeyMapLock.Lock()
		r.leaseIdToKeyMap[leaseId] = key
		r.leaseIdToKeyMapLock.Unlock()
	}
	id.current += req.Cost
	currentWithLeases += req.Cost
	ratelimitsCount.WithLabelValues("true").Inc()
	return RatelimitResponse{Pass: true, Remaining: req.Limit - currentWithLeases, Reset: id.reset.UnixMilli(), Limit: req.Limit, Current: currentWithLeases}
}

func (r *slidingWindow) SetCurrent(ctx context.Context, req SetCurrentRequest) error {
	ctx, span := tracing.Start(ctx, "slidingWindow.SetCurrent")
	defer span.End()
	key := BuildKey(req.Identifier, req.Duration)

	r.identifiersLock.RLock(ctx)
	id, ok := r.identifiers[req.Identifier]
	r.identifiersLock.RUnlock(ctx)
	span.SetAttributes(attribute.Bool("identifierExisted", ok))

	if !ok {
		reset := time.UnixMilli((time.Now().UnixMilli()/req.Duration.Milliseconds() + 1) * req.Duration.Milliseconds())
		id = newIdentifierWindow(key, reset)
		r.identifiers[req.Identifier] = id
	}

	// Only increment the current value if the new value is greater than the current value
	// Due to varing network latency, we may receive out of order responses and could decrement the
	// current value, which would result in inaccurate rate limiting
	id.Lock()
	defer id.Unlock()
	if req.Current > id.current {
		id.current = req.Current
	}

	return nil
}

func (r *slidingWindow) CommitLease(ctx context.Context, req CommitLeaseRequest) error {
	ctx, span := tracing.Start(ctx, "slidingWindow.SetCurrent")
	defer span.End()

	r.leaseIdToKeyMapLock.RLock()
	key, ok := r.leaseIdToKeyMap[req.LeaseId]
	r.leaseIdToKeyMapLock.RUnlock()
	if !ok {
		r.logger.Warn().Str("leaseId", req.LeaseId).Msg("leaseId not found")
		return nil
	}

	r.identifiersLock.Lock(ctx)
	defer r.identifiersLock.Unlock(ctx)
	window, ok := r.identifiers[key]
	if !ok {
		r.logger.Warn().Str("key", key).Msg("key not found")
		return nil
	}

	_, ok = window.leases[req.LeaseId]
	if !ok {
		r.logger.Warn().Str("leaseId", req.LeaseId).Msg("leaseId not found")
		return nil
	}

	return nil

}
