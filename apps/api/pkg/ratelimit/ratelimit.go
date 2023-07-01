package ratelimit

import (
	"sync"
	"time"
)

type RatelimitRequest struct {
	Identifier     string
	Max            int64
	RefillRate     int64
	RefillInterval int64
}

type RatelimitResponse struct {
	Pass      bool
	Limit     int64
	Remaining int64
	Reset     int64
}

type bucket struct {
	sync.RWMutex

	// unix milli of when this bucket was created
	startTime int64
	// Currently remaining tokens
	remaining int64

	// how many tokens at maximum fill
	max int64

	// how many tokens to refill per interval
	refillRate int64
	// in milliseconds
	refillInterval int64

	// the window where the last refill happened
	lastTick int64
}

func newBucket(refillRate int64, refillInterval int64, max int64) *bucket {
	now := time.Now().UnixMilli()
	return &bucket{
		startTime:      now,
		remaining:      max,
		max:            max,
		refillRate:     refillRate,
		refillInterval: refillInterval,
		lastTick:       0,
	}
}

func (b *bucket) take() RatelimitResponse {
	now := time.Now().UnixMilli()

	// The number of the window since bucket creation
	tick := (now - b.startTime) / b.refillInterval

	reset := b.startTime + ((tick + 1) * b.refillInterval)

	b.Lock()
	defer b.Unlock()

	if b.lastTick < tick {
		b.remaining += (tick - b.lastTick) * b.refillRate
		if b.remaining > b.max {
			b.remaining = b.max
		}
		b.lastTick = tick
	}

	if b.remaining <= 0 {
		return RatelimitResponse{
			Pass:      false,
			Limit:     b.max,
			Remaining: 0,
			Reset:     reset,
		}
	}

	b.remaining -= 1
	return RatelimitResponse{
		Pass:      true,
		Limit:     b.max,
		Remaining: b.remaining,
		Reset:     reset,
	}

}

type Ratelimiter struct {
	stateLock sync.RWMutex
	state     map[string]*bucket
}

func New() *Ratelimiter {

	r := &Ratelimiter{
		stateLock: sync.RWMutex{},
		state:     make(map[string]*bucket),
	}

	go func() {
		for range time.NewTicker(time.Minute).C {
			now := time.Now().UnixMilli()
			r.stateLock.Lock()

			for id, b := range r.state {
				b.Lock()
				currentTick := (now - b.startTime) / b.refillInterval
				requiredTicksToRefill := (b.max - b.remaining) / b.refillRate

				if requiredTicksToRefill > currentTick-b.lastTick {
					delete(r.state, id)
				}
				b.Unlock()
			}
			r.stateLock.Unlock()

		}
	}()

	return r

}

func (r *Ratelimiter) Take(req RatelimitRequest) RatelimitResponse {
	r.stateLock.RLock()

	b, ok := r.state[req.Identifier]
	r.stateLock.RUnlock()
	if ok {
		return b.take()
	}

	r.stateLock.Lock()
	// Check again since we are in a new lock and another goroutine could have created it now
	b, ok = r.state[req.Identifier]
	if ok {
		r.stateLock.Unlock()
		return b.take()
	}

	b = newBucket(req.RefillRate, req.RefillInterval, req.Max)
	r.state[req.Identifier] = b
	r.stateLock.Unlock()

	return b.take()

}
