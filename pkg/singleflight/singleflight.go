package singleflight

import (
	"context"
	"sync"
)

// Group deduplicates concurrent calls for the same key. Calls for different
// keys proceed independently. Groups must be constructed with [New] and are safe
// for concurrent access.
type Group[K comparable, V any] struct {
	// The mutex protects calls and result publication.
	mutex sync.Mutex
	// Calls contains active calls indexed by key. A call is removed before its
	// done channel closes so a retry cannot join a completed call.
	calls map[K]*call[V]
}

// New returns a Group with no active calls.
func New[K comparable, V any]() *Group[K, V] {
	return &Group[K, V]{
		mutex: sync.Mutex{},
		calls: make(map[K]*call[V]),
	}
}

// Do runs function once for key and returns the result to concurrent callers
// waiting on the same key. A waiting caller can stop waiting when its context
// is canceled without canceling the active call.
func (g *Group[K, V]) Do(
	ctx context.Context,
	key K,
	function func(context.Context) (V, error),
) (V, error) {
	if g == nil || g.calls == nil {
		panic("singleflight: Group must be constructed with New")
	}

	// If the executing caller is canceled or panics, a waiter retries with its
	// own function and context rather than reusing an incomplete result.
	for {
		if err := ctx.Err(); err != nil {
			var zero V
			return zero, err
		}

		activeCall, shouldExecute := g.acquire(key)
		if shouldExecute {
			functionReturned := false
			// A panic must release the key and wake waiters before propagating.
			defer func() {
				if !functionReturned {
					var zero V
					g.finish(key, activeCall, zero, nil, true)
				}
			}()

			value, err := function(ctx)
			functionReturned = true
			g.finish(key, activeCall, value, err, ctx.Err() != nil)
			return value, err
		}

		// Another call for this key is already in progress. Wait for its result
		// or stop when this caller's context is canceled.
		select {
		case <-activeCall.done:
			if activeCall.retry {
				continue
			}
			return activeCall.value, activeCall.err
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		}
	}
}

// call represents one in-flight call and its published result. Waiters must
// read retry, value, and err only after done closes.
type call[V any] struct {
	// Done closes after retry, value, and err have been published.
	done chan struct{}
	// Retry tells waiters to execute their own function because this call was
	// canceled or panicked.
	retry bool
	// Value is shared by callers waiting for the same key.
	value V
	// Err is the error returned with value.
	err error
}

// acquire returns the active call for key. If no call exists, it registers one
// and instructs the caller to execute the function. Otherwise, the caller waits
// for the returned call.
func (g *Group[K, V]) acquire(key K) (activeCall *call[V], shouldExecute bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if activeCall, ok := g.calls[key]; ok {
		return activeCall, false
	}

	var zero V
	activeCall = &call[V]{
		done:  make(chan struct{}),
		retry: false,
		value: zero,
		err:   nil,
	}
	g.calls[key] = activeCall
	return activeCall, true
}

// finish publishes a call result, removes its reservation, and releases its
// waiters. Callers must invoke finish exactly once for each reserved call.
func (g *Group[K, V]) finish(
	key K,
	activeCall *call[V],
	value V,
	err error,
	retry bool,
) {
	g.mutex.Lock()
	if g.calls[key] != activeCall {
		g.mutex.Unlock()
		panic("singleflight: cannot finish an unregistered call")
	}
	activeCall.retry = retry
	activeCall.value = value
	activeCall.err = err
	delete(g.calls, key)
	close(activeCall.done)
	g.mutex.Unlock()
}
