package circuitbreaker

import "errors"

var (
	// ErrTripped is returned when the circuit breaker is in the [Open] state and
	// is rejecting requests to protect the downstream service. Callers should
	// implement fallback logic or return an appropriate error to their clients.
	ErrTripped = errors.New("circuit breaker is open")

	// ErrTooManyRequests is returned when the circuit breaker is in the [HalfOpen]
	// state and has already allowed the maximum number of probe requests through.
	// The caller should wait for the current probe requests to complete before
	// retrying.
	ErrTooManyRequests = errors.New("too many requests during half open state")
)

// IsErrTripped reports whether err is or wraps [ErrTripped].
func IsErrTripped(err error) bool {
	return errors.Is(err, ErrTripped)
}

// IsErrTooManyRequests reports whether err is or wraps [ErrTooManyRequests].
func IsErrTooManyRequests(err error) bool {
	return errors.Is(err, ErrTooManyRequests)
}
