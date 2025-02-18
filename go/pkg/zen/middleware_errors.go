package zen

import (
	"net/http"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func WithErrorHandling() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(s *Session) error {
			err := next(s)

			if err == nil {
				return nil
			}

			switch fault.GetTag(err) {
			case fault.NOT_FOUND:
				return s.JSON(http.StatusNotFound, api.NotFoundError{
					Title:     "Not Found",
					Type:      "https://unkey.com/docs/errors/not_found",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    s.responseStatus,
					Instance:  nil,
				})

			case fault.BAD_REQUEST:
				return s.JSON(http.StatusBadRequest, api.BadRequestError{
					Title:     "Bad Request",
					Type:      "https://unkey.com/docs/errors/bad_request",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    http.StatusBadRequest,
					Instance:  nil,
					Errors:    []api.ValidationError{},
				})

			case fault.UNAUTHORIZED:
				return s.JSON(http.StatusUnauthorized, api.UnauthorizedError{
					Title:     "Unauthorized",
					Type:      "https://unkey.com/docs/errors/unauthorized",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    s.responseStatus,
					Instance:  nil,
				})
			case fault.FORBIDDEN:
				return s.JSON(http.StatusForbidden, api.ForbiddenError{
					Title:     "Forbidden",
					Type:      "https://unkey.com/docs/errors/forbidden",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    s.responseStatus,
					Instance:  nil,
				})
			case fault.PROTECTED_RESOURCE:
				return s.JSON(http.StatusPreconditionFailed, api.PreconditionFailedError{
					Title:     "Resource is protected",
					Type:      "https://unkey.com/docs/errors/deletion_prevented",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    s.responseStatus,
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

			return s.JSON(http.StatusInternalServerError, api.InternalServerError{
				Title:     "Internal Server Error",
				Type:      "https://unkey.com/docs/errors/internal_server_error",
				Detail:    fault.UserFacingMessage(err),
				RequestId: s.requestID,
				Status:    s.responseStatus,
				Instance:  nil,
			})
		}
	}
}
