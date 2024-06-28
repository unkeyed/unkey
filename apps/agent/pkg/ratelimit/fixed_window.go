package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type identifierWindow struct {
	// unix milli timestamp of the start of the window
	id string

	current int64

	reset time.Time
}

type fixedWindow struct {
	identifiersLock sync.RWMutex
	identifiers     map[string]*identifierWindow
	logger          logging.Logger
}

func NewFixedWindow(logger logging.Logger) *fixedWindow {

	r := &fixedWindow{
		identifiersLock: sync.RWMutex{},
		identifiers:     make(map[string]*identifierWindow),
		logger:          logger,
	}

	go func() {
		for range time.NewTicker(time.Minute).C {
			now := time.Now()
			r.identifiersLock.Lock()

			for _, identifier := range r.identifiers {
				if identifier.reset.After(now) {
					delete(r.identifiers, identifier.id)
				}
			}
			r.identifiersLock.Unlock()
		}
	}()
	return r

}

func buildKey(identifier string, limit int64, duration int64) string {
	window := time.Now().UnixMilli() / duration
	return fmt.Sprintf("ratelimit:%s:%d:%d", identifier, limit, window)
}

func (r *fixedWindow) Take(req RatelimitRequest) RatelimitResponse {
	start := time.Now()
	key := buildKey(req.Identifier, req.Max, req.RefillInterval)
	defer func() {
		r.logger.Info().Str("key", key).Int64("latency", time.Since(start).Milliseconds()).Msg("fixedWindow.Take")
	}()

	r.identifiersLock.Lock()
	defer r.identifiersLock.Unlock()

	id, ok := r.identifiers[key]
	if !ok {
		id = &identifierWindow{id: key, current: 0, reset: time.Now().Add(time.Duration(req.RefillInterval) * time.Millisecond)}
		r.identifiers[key] = id
	}

	if id.current+req.Cost > req.Max {
		return RatelimitResponse{Pass: false, Remaining: req.Max - id.current, Reset: id.reset.UnixMilli(), Limit: req.Max, Current: id.current}
	}

	id.current += req.Cost
	return RatelimitResponse{Pass: true, Remaining: req.Max - id.current, Reset: id.reset.UnixMilli(), Limit: req.Max, Current: id.current}
}

func (r *fixedWindow) SetCurrent(req SetCurrentRequest) error {
	key := buildKey(req.Identifier, req.Max, req.RefillInterval)

	r.identifiersLock.Lock()
	defer r.identifiersLock.Unlock()

	id, ok := r.identifiers[req.Identifier]
	if !ok {
		id = &identifierWindow{id: key, current: 0, reset: time.Now().Add(time.Duration(req.RefillInterval) * time.Millisecond)}
		r.identifiers[req.Identifier] = id
	}
	id.current = req.Current

	r.logger.Info().Str("key", key).Bool("overwriting", ok).Msg("SetCurrent")
	return nil
}
