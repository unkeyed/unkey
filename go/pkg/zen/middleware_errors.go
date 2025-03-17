package zen

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// WithErrorHandling returns middleware that translates errors into appropriate
// HTTP responses. It uses status codes based on error tags:
//
//   - NOT_FOUND: 404 Not Found
//   - BAD_REQUEST: 400 Bad Request
//   - UNAUTHORIZED: 401 Unauthorized
//   - FORBIDDEN: 403 Forbidden
//   - PROTECTED_RESOURCE: 412 Precondition Failed
//   - Other errors: 500 Internal Server Error
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithErrorHandling()},
//	    route,
//	)
func WithErrorHandling(logger logging.Logger) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			err := next(ctx, s)

			if err == nil {
				return nil
			}

			//	errorSteps := fault.Flatten(err)
			//	if len(errorSteps) > 0 {

			//		var b strings.Builder
			//		b.WriteString("Error trace:\n")

			//		for i, step := range errorSteps {
			//			// Skip empty messages
			//			if step.Message == "" {
			//				continue
			//			}

			//			b.WriteString(fmt.Sprintf("  Step %d:\n", i+1))

			//			if step.Location != "" {
			//				b.WriteString(fmt.Sprintf("    Location: %s\n", step.Location))
			//			} else {
			//				b.WriteString("    Location: unknown\n")
			//			}

			//			b.WriteString(fmt.Sprintf("    Message: %s\n", step.Message))

			//			// Add a small separator between steps
			//			if i < len(errorSteps)-1 {
			//				b.WriteString("\n")
			//			}
			//		}

			//		logger.Error("api encountered errors", "trace", b.String())

			//	}

			switch fault.GetTag(err) {
			case fault.NOT_FOUND:
				return s.JSON(http.StatusNotFound, openapi.NotFoundError{
					Title:     "Not Found",
					Type:      "https://unkey.com/docs/errors/not_found",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusNotFound,
					Instance:  nil,
				})

			case fault.BAD_REQUEST:
				return s.JSON(http.StatusBadRequest, openapi.BadRequestError{
					Title:     "Bad Request",
					Type:      "https://unkey.com/docs/errors/bad_request",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusBadRequest,
					Instance:  nil,
					Errors:    []openapi.ValidationError{},
				})

			case fault.UNAUTHORIZED:
				return s.JSON(http.StatusUnauthorized, openapi.UnauthorizedError{
					Title:     "Unauthorized",
					Type:      "https://unkey.com/docs/errors/unauthorized",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusUnauthorized,
					Instance:  nil,
				})
			case fault.FORBIDDEN:
				return s.JSON(http.StatusForbidden, openapi.ForbiddenError{
					Title:     "Forbidden",
					Type:      "https://unkey.com/docs/errors/forbidden",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusForbidden,
					Instance:  nil,
				})
			case fault.INSUFFICIENT_PERMISSIONS:
				return s.JSON(http.StatusForbidden, openapi.ForbiddenError{
					Title:     "Insufficient Permissions",
					Type:      "https://unkey.com/docs/errors/insufficient_permissions",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusForbidden,
					Instance:  nil,
				})
			case fault.PROTECTED_RESOURCE:
				return s.JSON(http.StatusPreconditionFailed, openapi.PreconditionFailedError{
					Title:     "Resource is protected",
					Type:      "https://unkey.com/docs/errors/deletion_prevented",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusPreconditionFailed,
					Instance:  nil,
				})

			case fault.DATABASE_ERROR:
				break // fall through to default 500

			case fault.UNTAGGED:
				break // fall through to default 500

			case fault.ASSERTION_FAILED:
				break // fall through to default 500
			case fault.INTERNAL_SERVER_ERROR:
				break
			}

			return s.JSON(http.StatusInternalServerError, openapi.InternalServerError{
				Title:     "Internal Server Error",
				Type:      "https://unkey.com/docs/errors/internal_server_error",
				Detail:    fault.UserFacingMessage(err),
				RequestId: s.requestID,
				Status:    http.StatusInternalServerError,
				Instance:  nil,
			})
		}
	}
}
