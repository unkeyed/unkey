package zen

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// withErrorHandling returns a lightweight error handler exclusively for testing.
// only handles the specific error codes that zen package tests need to verify.
func withErrorHandling() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			err := next(ctx, s)
			if err == nil {
				return nil
			}

			// Store the internal error message for metrics logging before we
			// convert it to an HTTP response and lose the details.
			s.SetInternalError(fault.InternalMessage(err))

			// Get the error URN from the error
			urn, ok := fault.GetCode(err)
			if !ok {
				urn = codes.App.Internal.UnexpectedError.URN()
			}

			code, parseErr := codes.ParseURN(urn)
			if parseErr != nil {
				logger.Error("failed to parse error code", "error", parseErr.Error())
				code = codes.App.Internal.UnexpectedError
			}

			// nolint: exhaustive // Maps only the error codes used in zen tests
			switch urn {
			// Not Found errors
			case codes.UnkeyDataErrorsKeyNotFound:
				return s.JSON(http.StatusNotFound, openapi.NotFoundErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Not Found",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusNotFound,
					},
				})

			// Request Timeout errors
			case codes.UserErrorsBadRequestRequestTimeout:
				return s.JSON(http.StatusRequestTimeout, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Request Timeout",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusRequestTimeout,
						Errors: []openapi.ValidationError{},
					},
				})

			// Client Closed Request errors (499 - non-standard but widely used)
			case codes.UserErrorsBadRequestClientClosedRequest:
				return s.JSON(499, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Client Closed Request",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: 499,
						Errors: []openapi.ValidationError{},
					},
				})

			// Request Entity Too Large errors
			case codes.UserErrorsBadRequestRequestBodyTooLarge:
				return s.JSON(http.StatusRequestEntityTooLarge, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Request Entity Too Large",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusRequestEntityTooLarge,
						Errors: []openapi.ValidationError{},
					},
				})

			// Bad Request errors - Unreadable body
			case codes.UserErrorsBadRequestRequestBodyUnreadable:
				return s.JSON(http.StatusBadRequest, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Bad Request",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusBadRequest,
						Errors: []openapi.ValidationError{},
					},
				})

			// Internal errors - fall through to default
			default:
				logger.Error("unhandled error code in test handler", "urn", urn, "error", err.Error())
			}

			// Default: Internal Server Error
			return s.JSON(http.StatusInternalServerError, openapi.InternalServerErrorResponse{
				Meta: openapi.Meta{
					RequestId: s.RequestID(),
				},
				Error: openapi.BaseError{
					Title:  "Internal Server Error",
					Type:   code.DocsURL(),
					Detail: fault.UserFacingMessage(err),
					Status: http.StatusInternalServerError,
				},
			})
		}
	}
}
