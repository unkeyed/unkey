// Package wait holds shared polling and retry helpers used by multiple
// probes. Every helper here is deliberately small: a concrete probe
// almost always wants a single call that returns the observed state or
// a context-deadline error.
//
// Named "wait" rather than "assert" because pkg/assert already holds the
// general-purpose assertion library; the two must not be confused.
package wait

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrDeadline is returned when a Poll helper exhausts its budget
// without the condition becoming true. Callers should match with
// errors.Is so wrappers like fmt.Errorf("%w: details", err) are
// detected.
var ErrDeadline = errors.New("wait: deadline exceeded")

// Condition is the poll predicate.
//
//	(result, true, nil)  -> success; the caller receives result.
//	(_, false, nil)      -> keep polling.
//	(_, _, err)          -> abort with err (returned verbatim).
//
// A Condition that always returns (_, false, nil) will be driven to
// ErrDeadline by the caller's context deadline.
type Condition[T any] func(ctx context.Context) (T, bool, error)

// Poll invokes fn on a ticker of period until it returns done=true,
// returns an error, or ctx's deadline is reached.
//
// The first invocation happens immediately (before waiting a period),
// so a Condition that is already satisfied returns without sleeping.
// If ctx is already cancelled when Poll is called, Poll returns
// ErrDeadline without invoking fn at all.
//
// Period must be positive; zero or negative values panic because
// they are programmer errors, not conditions a caller can recover
// from.
func Poll[T any](ctx context.Context, period time.Duration, fn Condition[T]) (T, error) {
	var zero T

	if period <= 0 {
		panic(fmt.Sprintf("wait.Poll: period must be positive, got %v", period))
	}

	// Fast-fail on an already-cancelled context so callers do not see
	// a spurious first invocation when they have already given up.
	if err := ctx.Err(); err != nil {
		return zero, fmt.Errorf("%w: %w", ErrDeadline, err)
	}

	// Immediate first attempt: many conditions are already true, and
	// the probes that rely on this helper should not pay a period of
	// latency for the common case.
	v, done, err := fn(ctx)
	if err != nil {
		return zero, err
	}

	if done {
		return v, nil
	}

	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("%w: %w", ErrDeadline, ctx.Err())
		case <-t.C:
			v, done, err := fn(ctx)
			if err != nil {
				return zero, err
			}
			if done {
				return v, nil
			}
		}
	}
}
