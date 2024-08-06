package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/mutex"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type identifierWindow struct {
	id string

	current int64

	reset time.Time
}

type fixedWindow struct {
	identifiersLock *mutex.TraceLock
	identifiers     map[string]*identifierWindow
	logger          logging.Logger
}

func NewFixedWindow(logger logging.Logger) *fixedWindow {

	r := &fixedWindow{
		identifiersLock: mutex.New(),
		identifiers:     make(map[string]*identifierWindow),
		logger:          logger,
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

func buildKey(identifier string, limit int64, duration int64) string {
	window := time.Now().UnixMilli() / duration
	return fmt.Sprintf("ratelimit:%s:%d:%d", identifier, limit, window)
}

func (r *fixedWindow) Take(ctx context.Context, req RatelimitRequest) RatelimitResponse {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("fixedWindow.Take", req.Name))
	defer span.End()

	key := buildKey(req.Identifier, req.Max, req.RefillInterval)
	span.SetAttributes(attribute.String("key", key))

	r.identifiersLock.Lock(ctx)
	defer r.identifiersLock.Unlock(ctx)

	id, ok := r.identifiers[key]
	span.SetAttributes(attribute.Bool("identifierExisted", ok))
	if !ok {
		reset := time.UnixMilli((time.Now().UnixMilli()/req.RefillInterval + 1) * req.RefillInterval)
		id = &identifierWindow{id: key, current: 0, reset: reset}
		r.identifiers[key] = id
	}

	if id.current+req.Cost > req.Max {
		ratelimitsRejected.Inc()
		return RatelimitResponse{Pass: false, Remaining: req.Max - id.current, Reset: id.reset.UnixMilli(), Limit: req.Max, Current: id.current}
	}

	id.current += req.Cost
	ratelimitsPassed.Inc()
	return RatelimitResponse{Pass: true, Remaining: req.Max - id.current, Reset: id.reset.UnixMilli(), Limit: req.Max, Current: id.current}
}

func (r *fixedWindow) SetCurrent(ctx context.Context, req SetCurrentRequest) error {
	ctx, span := tracing.Start(ctx, "fixedWindow.SetCurrent")
	defer span.End()
	key := buildKey(req.Identifier, req.Max, req.RefillInterval)

	r.identifiersLock.Lock(ctx)
	defer r.identifiersLock.Unlock(ctx)
	id, ok := r.identifiers[req.Identifier]
	span.SetAttributes(attribute.Bool("identifierExisted", ok))

	if !ok {
		reset := time.UnixMilli((time.Now().UnixMilli()/req.RefillInterval + 1) * req.RefillInterval)
		id = &identifierWindow{id: key, current: 0, reset: reset}
		r.identifiers[req.Identifier] = id
	}

	// Only increment the current value if the new value is greater than the current value
	// Due to varing network latency, we may receive out of order responses and could decrement the
	// current value, which would result in inaccurate rate limiting
	if req.Current > id.current {
		id.current = req.Current
	} else if req.Current < id.current {
		r.logger.Debug().Int64("current", id.current).Int64("req", req.Current).Msg("Ignoring SetCurrent request cause it's lower than current")
	}

	r.logger.Debug().Str("key", key).Bool("overwriting", ok).Msg("SetCurrent")
	return nil
}
