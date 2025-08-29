package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// WithPanicRecovery returns middleware that recovers from panics and converts them
// into appropriate HTTP error responses.
func WithPanicRecovery(logger logging.Logger) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Get stack trace
					stack := debug.Stack()

					// Log the panic with stack trace
					logger.Error("panic recovered",
						"panic", fmt.Sprintf("%v", r),
						"requestId", s.RequestID(),
						"method", s.r.Method,
						"path", s.r.URL.Path,
						"stack", string(stack),
					)

					metrics.PanicsTotal.WithLabelValues("", s.r.URL.Path).Inc()

					// Convert panic to an error
					panicErr := fault.New("Internal Server Error",
						fault.Code(codes.App.Internal.UnexpectedError.URN()),
						fault.Internal(fmt.Sprintf("panic: %v", r)),
						fault.Public("An unexpected error occurred while processing your request."),
					)

					// Return error response
					err = s.JSON(http.StatusInternalServerError, ErrorResponse{
						Meta: Meta{
							RequestID: s.RequestID(),
						},
						Error: ErrorDetail{
							Title:  "Internal Server Error",
							Type:   codes.App.Internal.UnexpectedError.DocsURL(),
							Detail: fault.UserFacingMessage(panicErr),
							Status: http.StatusInternalServerError,
						},
					})
				}
			}()

			return next(ctx, s)
		}
	}
}
