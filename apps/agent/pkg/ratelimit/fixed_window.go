package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/mutex"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"go.opentelemetry.io/otel/attribute"
)

type lease struct {
	id        string
	cost      int64
	expiresAt time.Time

	// > 0 if the lease is committed
	committedTokens int64
}

type identifierWindow struct {
	sync.Mutex
	id string

	current int64

	reset time.Time

	// leaseId -> lease
	leases map[string]lease
}

func newIdentifierWindow(id string, reset time.Time) *identifierWindow {
	return &identifierWindow{
		id:      id,
		current: 0,
		reset:   reset,
		leases:  make(map[string]lease),
	}
}

type fixedWindow struct {
	identifiersLock     *mutex.TraceLock
	identifiers         map[string]*identifierWindow
	leaseIdToKeyMapLock sync.RWMutex
	// Store a reference leaseId -> window key
	leaseIdToKeyMap map[string]string
	logger          logging.Logger
}

func NewFixedWindow(logger logging.Logger) *fixedWindow {

	r := &fixedWindow{
		identifiersLock:     mutex.New(),
		identifiers:         make(map[string]*identifierWindow),
		leaseIdToKeyMapLock: sync.RWMutex{},
		leaseIdToKeyMap:     make(map[string]string),
		logger:              logger,
	}

	repeat.Every(time.Minute, r.removeExpiredIdentifiers)
	return r

}

func (r *fixedWindow) removeExpiredIdentifiers() {
	ctx := context.Background()
	r.identifiersLock.Lock(ctx)
	defer r.identifiersLock.Unlock(ctx)

	activeRatelimits.Set(float64(len(r.identifiers)))
	now := time.Now()
	for _, identifier := range r.identifiers {
		if identifier.reset.After(now) {
			delete(r.identifiers, identifier.id)
		}
	}
}

func buildKey(identifier string, limit int64, duration time.Duration) string {
	window := time.Now().UnixMilli() / duration.Milliseconds()
	return fmt.Sprintf("ratelimit:%s:%d:%d", identifier, limit, window)
}

func (r *fixedWindow) Take(ctx context.Context, req RatelimitRequest) RatelimitResponse {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("fixedWindow.Take", req.Name))
	defer span.End()

	key := buildKey(req.Identifier, req.Limit, req.Duration)
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

func (r *fixedWindow) SetCurrent(ctx context.Context, req SetCurrentRequest) error {
	ctx, span := tracing.Start(ctx, "fixedWindow.SetCurrent")
	defer span.End()
	key := buildKey(req.Identifier, req.Limit, req.Duration)

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

func (r *fixedWindow) CommitLease(ctx context.Context, req CommitLeaseRequest) error {
	ctx, span := tracing.Start(ctx, "fixedWindow.SetCurrent")
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
