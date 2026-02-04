package zen

import (
	"context"
	"errors"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	// DefaultRequestTimeout is the default timeout for API requests
	DefaultRequestTimeout = 30 * time.Second
)

// WithTimeout returns middleware that enforces a timeout on request processing.
// It differentiates between client-initiated cancellations and server-side timeouts.
func WithTimeout(timeout time.Duration) Middleware {
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}

	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			// Create a new context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Process the request
			err := next(timeoutCtx, s)
			if err == nil {
				return nil
			}

			isTimeoutErr := errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
			if !isTimeoutErr {
				// Not a timeout error, pass it through
				return err
			}

			// For context errors, always reclassify them properly regardless of any existing code.
			// This handles cases where application code (e.g., DB operations) wrapped a context error
			// with a service-specific code like ServiceUnavailable. Context cancellation should always
			// be classified as either ClientClosedRequest or RequestTimeout, not as a server error.

			// Check if the original request context was canceled (client closed connection)
			if ctx.Err() != nil {
				// Client closed the request
				return fault.Wrap(err,
					fault.Code(codes.User.BadRequest.ClientClosedRequest.URN()),
					fault.Internal("The client closed the connection before the request completed"),
					fault.Public("Client closed request"),
				)
			}

			// Server-side timeout, we took too long to process the request
			return fault.Wrap(err,
				fault.Code(codes.User.BadRequest.RequestTimeout.URN()),
				fault.Internal("The request exceeded the maximum processing time"),
				fault.Public("Request timeout"),
			)
		}
	}
}
