package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

type bucket struct {
	sync.RWMutex

	// unix milli of when this bucket was created
	startTime int64
	// Currently remaining tokens
	remaining int32

	// how many tokens at maximum fill
	max int32

	// how many tokens to refill per interval
	refillRate int32
	// in milliseconds
	refillInterval int32

	// the window where the last refill happened
	lastTick int64
}

func newBucket(refillRate int32, refillInterval int32, max int32) *bucket {
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

func (b *bucket) take(tokens int32) RatelimitResponse {
	now := time.Now().UnixMilli()

	// The number of the window since bucket creation
	tick := (now - b.startTime) / int64(b.refillInterval)

	reset := b.startTime + ((tick + 1) * int64(b.refillInterval))

	b.Lock()
	defer b.Unlock()

	// if b.lastTick < tick {
	// 	b.remaining += int32((tick - b.lastTick) * int64(b.refillRate))
	// 	if b.remaining > b.max {
	// 		b.remaining = b.max
	// 	}
	// 	b.lastTick = tick
	// }

	if b.remaining-tokens < 0 {
		return RatelimitResponse{
			Pass:      false,
			Limit:     b.max,
			Remaining: b.remaining,
			Reset:     reset,
		}
	}
	fmt.Println("tokens", tokens)
	fmt.Printf("b 1: %+v\n", b)

	b.remaining -= tokens
	fmt.Printf("b 2: %+v\n", b)

	return RatelimitResponse{
		Pass:      true,
		Limit:     b.max,
		Remaining: b.remaining,
		Reset:     reset,
	}

}

type inMemory struct {
	stateLock sync.RWMutex
	state     map[string]*bucket
}

func NewInMemory() *inMemory {

	r := &inMemory{
		stateLock: sync.RWMutex{},
		state:     make(map[string]*bucket),
	}

	go func() {
		for range time.NewTicker(time.Minute).C {
			now := time.Now().UnixMilli()
			r.stateLock.Lock()

			for id, b := range r.state {
				b.Lock()
				currentTick := (now - b.startTime) / int64(b.refillInterval)
				requiredTicksToRefill := (b.max - b.remaining) / b.refillRate

				if int64(requiredTicksToRefill) > currentTick-b.lastTick {
					delete(r.state, id)
				}
				b.Unlock()
			}
			r.stateLock.Unlock()

		}
	}()

	return r

}

func (r *inMemory) Take(req RatelimitRequest) RatelimitResponse {
	fmt.Printf("req: %+v\n", req)

	r.stateLock.RLock()

	b, ok := r.state[req.Identifier]
	r.stateLock.RUnlock()
	if ok {
		return b.take(req.Cost)
	}

	r.stateLock.Lock()
	// Check again since we are in a new lock and another goroutine could have created it now
	b, ok = r.state[req.Identifier]
	if ok {
		r.stateLock.Unlock()
		return b.take(req.Cost)
	}

	b = newBucket(req.RefillRate, req.RefillInterval, req.Max)
	r.state[req.Identifier] = b
	r.stateLock.Unlock()

	return b.take(req.Cost)

}
