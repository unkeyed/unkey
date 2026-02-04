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

			// Client disconnected - this takes precedence over any other error classification
			if ctx.Err() != nil {
				return fault.Wrap(err,
					fault.Code(codes.User.BadRequest.ClientClosedRequest.URN()),
					fault.Internal("The client closed the connection before the request completed"),
					fault.Public("Client closed request"),
				)
			}

			// Server-side timeout - always reclassify regardless of any wrapped codes
			// App code may have wrapped context.DeadlineExceeded with ServiceUnavailable,
			// but it's still a timeout, not a 500
			if errors.Is(err, context.DeadlineExceeded) {
				return fault.Wrap(err,
					fault.Code(codes.User.BadRequest.RequestTimeout.URN()),
					fault.Internal("The request exceeded the maximum processing time"),
					fault.Public("Request timeout"),
				)
			}

			// Other errors pass through unchanged
			return err
		}
	}
}
