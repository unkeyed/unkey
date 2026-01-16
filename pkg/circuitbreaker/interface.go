package circuitbreaker

import (
	"context"
)

// State represents the current state of a circuit breaker. The circuit breaker
// transitions between states based on the success or failure of requests:
// [Closed] -> [Open] -> [HalfOpen] -> [Closed] (or back to [Open] on failure).
type State string

var (
	// Open indicates the circuit breaker is tripped and rejecting all requests
	// without calling the downstream service. The circuit remains open until
	// the timeout period elapses, after which it transitions to [HalfOpen].
	Open State = "open"

	// HalfOpen indicates the circuit breaker is testing whether the downstream
	// service has recovered. A limited number of probe requests are allowed
	// through. If they succeed, the circuit transitions to [Closed]; if any
	// fail, it returns to [Open].
	HalfOpen State = "halfopen"

	// Closed indicates the circuit breaker is operating normally and allowing
	// all requests through to the downstream service. If failures exceed the
	// trip threshold within a cyclic period, the circuit transitions to [Open].
	Closed State = "closed"
)

// CircuitBreaker wraps operations that call downstream services and provides
// automatic failure detection and recovery. When failures exceed a threshold,
// the circuit "opens" to fail fast and reduce load on the struggling service.
type CircuitBreaker[Res any] interface {
	// Do executes the provided function if the circuit is closed or allows a
	// probe request in half-open state. Returns [ErrTripped] if the circuit is
	// open, or [ErrTooManyRequests] if the half-open probe limit is exceeded.
	Do(ctx context.Context, fn func(context.Context) (Res, error)) (Res, error)
}
