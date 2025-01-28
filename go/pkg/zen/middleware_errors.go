package zen

import (
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
				return s.JSON(404, api.NotFoundError{
					Title:     "Not Found",
					Type:      "https://unkey.com/docs/errors/not_found",
					Detail:    fault.UserFacingMessage(err),
					RequestId: s.requestID,
					Status:    s.responseStatus,
					Instance:  nil,
				})

			case fault.DATABASE_ERROR:
				break // fall through to default 500

			case fault.UNTAGGED:
				break // fall through to default 500

			case fault.INTERNAL_SERVER_ERROR:
				break
			}

			return s.JSON(500, api.InternalServerError{
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
