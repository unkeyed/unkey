package circuitbreaker

import "errors"

var (
	ErrTripped         = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests during half open state")
)

func IsErrTripped(err error) bool {
	return errors.Is(err, ErrTripped)
}

func IsErrTooManyRequests(err error) bool {
	return errors.Is(err, ErrTooManyRequests)
}
