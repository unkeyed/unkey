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
			return g.execute(ctx, key, activeCall, function)
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

// DoAsync reserves key before passing its work to schedule. It returns true
// when work was scheduled and false when another call already reserved key.
// DoAsync must accept the work without executing it before returning.
func (g *Group[K, V]) DoAsync(
	ctx context.Context,
	key K,
	schedule func(func()),
	function func(context.Context) (V, error),
) bool {
	if g == nil || g.calls == nil {
		panic("singleflight: Group must be constructed with New")
	}
	if ctx.Err() != nil {
		return false
	}

	activeCall, shouldExecute := g.acquire(key)
	if !shouldExecute {
		return false
	}

	scheduled := false
	defer func() {
		if !scheduled {
			var zero V
			g.finish(key, activeCall, zero, nil, true)
		}
	}()
	schedule(func() {
		_, _ = g.execute(ctx, key, activeCall, function)
	})
	scheduled = true
	return true
}

// DoManyAsync reserves every currently inactive key before passing one batch
// to schedule. It returns true when at least one key was scheduled. The
// function must return a result for each key that it receives.
// DoManyAsync must accept the work without executing it before returning.
func (g *Group[K, V]) DoManyAsync(
	ctx context.Context,
	keys []K,
	schedule func(func()),
	function func(context.Context, []K) (map[K]V, error),
) bool {
	if g == nil || g.calls == nil {
		panic("singleflight: Group must be constructed with New")
	}
	if ctx.Err() != nil {
		return false
	}

	activeCalls, keysToExecute := g.acquireMany(keys)
	if len(keysToExecute) == 0 {
		return false
	}

	scheduled := false
	defer func() {
		if !scheduled {
			g.finishMany(activeCalls, nil, nil, true)
		}
	}()
	schedule(func() {
		g.executeMany(ctx, activeCalls, keysToExecute, function)
	})
	scheduled = true
	return true
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

// acquireMany reserves each unique key that has no active call. Holding one
// lock preserves the input batch against interleaved reservations.
func (g *Group[K, V]) acquireMany(keys []K) (map[K]*call[V], []K) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	activeCalls := make(map[K]*call[V], len(keys))
	keysToExecute := make([]K, 0, len(keys))
	seen := make(map[K]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := g.calls[key]; ok {
			continue
		}

		var zero V
		activeCall := &call[V]{
			done:  make(chan struct{}),
			retry: false,
			value: zero,
			err:   nil,
		}
		g.calls[key] = activeCall
		activeCalls[key] = activeCall
		keysToExecute = append(keysToExecute, key)
	}
	return activeCalls, keysToExecute
}

// execute runs one reserved call and always releases its waiters. A panic marks
// the result for retry before propagating to the executing caller.
func (g *Group[K, V]) execute(
	ctx context.Context,
	key K,
	activeCall *call[V],
	function func(context.Context) (V, error),
) (V, error) {
	if err := ctx.Err(); err != nil {
		var zero V
		g.finish(key, activeCall, zero, nil, true)
		return zero, err
	}

	functionReturned := false
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

// executeMany runs one reserved batch and always releases every key. A panic
// marks every result for retry before propagating to the executing caller.
func (g *Group[K, V]) executeMany(
	ctx context.Context,
	activeCalls map[K]*call[V],
	keys []K,
	function func(context.Context, []K) (map[K]V, error),
) {
	if ctx.Err() != nil {
		g.finishMany(activeCalls, nil, nil, true)
		return
	}

	functionReturned := false
	defer func() {
		if !functionReturned {
			g.finishMany(activeCalls, nil, nil, true)
		}
	}()

	values, err := function(ctx, keys)
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			panic("singleflight: batch function omitted a result")
		}
	}
	functionReturned = true
	g.finishMany(activeCalls, values, err, ctx.Err() != nil)
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

// finishMany publishes one batch, removes its reservations, and releases its
// waiters as one atomic state transition.
func (g *Group[K, V]) finishMany(
	activeCalls map[K]*call[V],
	values map[K]V,
	err error,
	retry bool,
) {
	g.mutex.Lock()
	for key, activeCall := range activeCalls {
		if g.calls[key] != activeCall {
			g.mutex.Unlock()
			panic("singleflight: cannot finish an unregistered call")
		}
	}
	for key, activeCall := range activeCalls {
		activeCall.retry = retry
		activeCall.value = values[key]
		activeCall.err = err
		delete(g.calls, key)
	}
	for _, activeCall := range activeCalls {
		close(activeCall.done)
	}
	g.mutex.Unlock()
}
