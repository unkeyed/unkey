package zen

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func WithPanicRecovery(m Metrics) Middleware {
	if m == nil {
		m = NoopMetrics{}
	}
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()

					logger.Error("panic recovered",
						"panic", fmt.Sprintf("%v", r),
						"requestId", s.RequestID(),
						"method", s.r.Method,
						"path", s.r.URL.Path,
						"stack", string(stack),
					)

					m.RecordPanic("", s.r.URL.Path)

					panicErr := fault.New("Internal Server Error",
						fault.Code(codes.App.Internal.UnexpectedError.URN()),
						fault.Internal(fmt.Sprintf("panic: %v", r)),
						fault.Public("An unexpected error occurred while processing your request."),
					)

					err = s.JSON(http.StatusInternalServerError, openapi.InternalServerErrorResponse{
						Meta: openapi.Meta{
							RequestId: s.RequestID(),
						},
						Error: openapi.BaseError{
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
