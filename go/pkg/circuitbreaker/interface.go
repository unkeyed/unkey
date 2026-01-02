package circuitbreaker

import (
	"context"
)

type State string

var (
	// Open state means the circuit breaker is open and requests are not allowed
	// to pass through
	Open State = "open"
	// HalfOpen state means the circuit breaker is in a state of testing the
	// upstream service to see if it has recovered
	HalfOpen State = "halfopen"
	// Closed state means the circuit breaker is allowing requests to pass
	// through to the upstream service
	Closed State = "closed"
)

type CircuitBreaker[Res any] interface {
	Do(ctx context.Context, fn func(context.Context) (Res, error)) (Res, error)
}
