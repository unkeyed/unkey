package circuitbreaker

import (
	"context"
	"errors"
)

// State represents the current operational state of a circuit breaker.
type State string

var (
	// Open state means the circuit breaker is open and requests are not allowed
	// to pass through. This state is activated when the failure threshold is
	// exceeded, preventing further requests to a failing service to allow it
	// time to recover.
	Open State = "open"

	// HalfOpen state means the circuit breaker is in a state of testing the
	// upstream service to see if it has recovered. In this state, a limited
	// number of requests are allowed through to test the service. If these
	// succeed, the circuit will close; if they fail, it will reopen.
	HalfOpen State = "halfopen"

	// Closed state means the circuit breaker is allowing requests to pass
	// through to the upstream service. This is the normal operational state
	// when the service is functioning correctly.
	Closed State = "closed"
)

var (
	// ErrTripped is returned when the circuit breaker is open and requests
	// are being blocked to protect the downstream service.
	ErrTripped = errors.New("circuit breaker is open")

	// ErrTooManyRequests is returned when too many requests have been made
	// during the half-open state, exceeding the configured test request limit.
	ErrTooManyRequests = errors.New("too many requests during half open state")
)

// CircuitBreaker provides a mechanism to detect failures and prevent
// cascading failures by blocking requests to failing dependencies when
// they exceed a threshold of failures.
//
// The generic parameter Res represents the successful response type from
// the protected operation.
type CircuitBreaker[Res any] interface {
	// Do executes the provided function with circuit breaker protection.
	// If the circuit is open, it will return ErrTripped without calling the function.
	// If the circuit is half-open and too many test requests are in flight,
	// it will return ErrTooManyRequests.
	//
	// The function is executed with the provided context, and the circuit breaker
	// tracks its success or failure to determine whether to trip or reset the circuit.
	Do(ctx context.Context, fn func(context.Context) (Res, error)) (Res, error)
}
