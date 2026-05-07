package zen

import (
	"context"
	"errors"
	"net/http"
	"strings"
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
//
// Protocol upgrades (e.g. WebSocket) bypass the timeout: a hijacked connection
// is no longer a "request" with a meaningful deadline, and cancelling its
// context tears down the bidirectional tunnel mid-session.
func WithTimeout(timeout time.Duration) Middleware {
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}

	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			if isProtocolUpgrade(s.Request()) {
				return next(ctx, s)
			}

			// Create a new context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Process the request
			err := next(timeoutCtx, s)
			if err == nil {
				return nil
			}

			// Client disconnected - the parent context was canceled (not deadline exceeded)
			if errors.Is(ctx.Err(), context.Canceled) {
				return fault.Wrap(err,
					fault.Code(codes.User.BadRequest.ClientClosedRequest.URN()),
					fault.Internal("The client closed the connection before the request completed"),
					fault.Public("Client closed request"),
				)
			}

			// Server-side timeout - always reclassify regardless of any wrapped codes.
			// App code may have wrapped context.DeadlineExceeded with ServiceUnavailable,
			// but it's still a timeout, not a 500.
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

// isProtocolUpgrade reports whether the request is asking the server to switch
// protocols (RFC 7230 §6.7). Both headers are required; Connection is a
// comma-separated token list and Upgrade values are case-insensitive.
func isProtocolUpgrade(r *http.Request) bool {
	if r == nil || r.Header.Get("Upgrade") == "" {
		return false
	}
	for _, token := range strings.Split(r.Header.Get("Connection"), ",") {
		if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
			return true
		}
	}
	return false
}
