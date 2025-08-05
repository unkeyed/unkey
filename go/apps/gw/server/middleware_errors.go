package server

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ErrorResponse is the standard error response format for the gateway.
type ErrorResponse struct {
	Meta  Meta        `json:"meta"`
	Error ErrorDetail `json:"error"`
}

// Meta contains metadata about the request.
type Meta struct {
	RequestID string `json:"requestId"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Title  string `json:"title"`
	Type   string `json:"type,omitempty"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

// WithErrorHandling returns middleware that translates errors into appropriate
// HTTP responses based on error URNs.
func WithErrorHandling(logger logging.Logger) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			err := next(ctx, s)
			if err == nil {
				return nil
			}

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

			// Determine status code based on error type
			status := http.StatusInternalServerError

			switch urn {
			// Not Found errors
			case codes.UnkeyDataErrorsKeyNotFound,
				codes.UnkeyDataErrorsWorkspaceNotFound,
				codes.UnkeyDataErrorsApiNotFound,
				codes.UnkeyDataErrorsPermissionNotFound,
				codes.UnkeyDataErrorsRoleNotFound,
				codes.UnkeyDataErrorsKeyAuthNotFound,
				codes.UnkeyDataErrorsRatelimitNamespaceNotFound,
				codes.UnkeyDataErrorsRatelimitOverrideNotFound,
				codes.UnkeyDataErrorsIdentityNotFound,
				codes.UnkeyDataErrorsAuditLogNotFound:
				status = http.StatusNotFound

			// Bad Request errors
			case codes.UnkeyAppErrorsValidationInvalidInput,
				codes.UnkeyAuthErrorsAuthenticationMissing,
				codes.UnkeyAuthErrorsAuthenticationMalformed,
				codes.UserErrorsBadRequestPermissionsQuerySyntaxError:
				status = http.StatusBadRequest

			// Unauthorized errors
			case codes.UnkeyAuthErrorsAuthenticationKeyNotFound:
				status = http.StatusUnauthorized

			// Forbidden errors
			case codes.UnkeyAuthErrorsAuthorizationForbidden,
				codes.UnkeyAuthErrorsAuthorizationInsufficientPermissions,
				codes.UnkeyAuthErrorsAuthorizationKeyDisabled,
				codes.UnkeyAuthErrorsAuthorizationWorkspaceDisabled:
				status = http.StatusForbidden

			// Conflict errors
			case codes.UnkeyDataErrorsIdentityDuplicate,
				codes.UnkeyDataErrorsRoleDuplicate,
				codes.UnkeyDataErrorsPermissionDuplicate:
				status = http.StatusConflict

			// Precondition Failed
			case codes.UnkeyAppErrorsProtectionProtectedResource,
				codes.UnkeyAppErrorsPreconditionPreconditionFailed:
				status = http.StatusPreconditionFailed
			}

			// Log the error
			logger.Error("gateway error",
				"error", err.Error(),
				"requestId", s.RequestID(),
				"publicMessage", fault.UserFacingMessage(err),
				"status", status,
			)

			// Create error response
			response := ErrorResponse{
				Meta: Meta{
					RequestID: s.RequestID(),
				},
				Error: ErrorDetail{
					Title:  http.StatusText(status),
					Type:   code.DocsURL(),
					Detail: fault.UserFacingMessage(err),
					Status: status,
				},
			}

			return s.JSON(status, response)
		}
	}
}
