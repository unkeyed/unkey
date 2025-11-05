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

			// Store the original error for metrics logging
			s.SetError(err)

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
			// Internal Server Error (500)
			case codes.UnkeyGatewayErrorsInternalInternalServerError,
				codes.UnkeyGatewayErrorsInternalKeyVerificationFailed,
				codes.UnkeyAppErrorsInternalUnexpectedError,
				codes.UnkeyAppErrorsInternalServiceUnavailable,
				codes.UnkeyAppErrorsValidationAssertionFailed,
				codes.UnkeyGatewayErrorsRoutingVMSelectionFailed:
				status = http.StatusInternalServerError

			// Bad Request (400)
			case codes.UnkeyGatewayErrorsValidationRequestInvalid,
				codes.UnkeyGatewayErrorsValidationResponseInvalid,
				codes.UnkeyAppErrorsValidationInvalidInput,
				codes.UnkeyAuthErrorsAuthenticationMissing,
				codes.UnkeyAuthErrorsAuthenticationMalformed,
				codes.UserErrorsBadRequestPermissionsQuerySyntaxError:
				status = http.StatusBadRequest

			// Unauthorized (401)
			case codes.UnkeyGatewayErrorsAuthUnauthorized,
				codes.UnkeyAuthErrorsAuthenticationKeyNotFound:
				status = http.StatusUnauthorized

			// Forbidden (403)
			case codes.UnkeyAuthErrorsAuthorizationForbidden,
				codes.UnkeyAuthErrorsAuthorizationInsufficientPermissions,
				codes.UnkeyAuthErrorsAuthorizationKeyDisabled,
				codes.UnkeyAuthErrorsAuthorizationWorkspaceDisabled:
				status = http.StatusForbidden

			// Not Found (404)
			case codes.UnkeyGatewayErrorsRoutingConfigNotFound,
				codes.UnkeyDataErrorsKeyNotFound,
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

			// Request Timeout (408)
			case codes.UserErrorsBadRequestRequestTimeout:
				status = http.StatusRequestTimeout

			// Conflict (409)
			case codes.UnkeyDataErrorsIdentityDuplicate,
				codes.UnkeyDataErrorsRoleDuplicate,
				codes.UnkeyDataErrorsPermissionDuplicate:
				status = http.StatusConflict

			// Gone (410)
			case codes.UnkeyDataErrorsRatelimitNamespaceGone:
				status = http.StatusGone

			// Precondition Failed (412)
			case codes.UnkeyAppErrorsProtectionProtectedResource,
				codes.UnkeyAppErrorsPreconditionPreconditionFailed:
				status = http.StatusPreconditionFailed

			// Request Entity Too Large (413)
			case codes.UserErrorsBadRequestRequestBodyTooLarge:
				status = http.StatusRequestEntityTooLarge

			// Too Many Requests (429)
			case codes.UnkeyGatewayErrorsAuthRateLimited:
				status = http.StatusTooManyRequests

			// Client Closed Request (499)
			case codes.UserErrorsBadRequestClientClosedRequest:
				status = 499

			// Bad Gateway (502)
			case codes.UnkeyGatewayErrorsProxyBadGateway,
				codes.UnkeyGatewayErrorsProxyProxyForwardFailed:
				status = http.StatusBadGateway

			// Service Unavailable (503)
			case codes.UnkeyGatewayErrorsProxyServiceUnavailable:
				status = http.StatusServiceUnavailable

			// Gateway Timeout (504)
			case codes.UnkeyGatewayErrorsProxyGatewayTimeout:
				status = http.StatusGatewayTimeout
			}

			// Log the error with correct status
			logger.Error("gateway error",
				"error", err.Error(),
				"requestId", s.RequestID(),
				"publicMessage", fault.UserFacingMessage(err),
				"status", status,
			)

			// Handle gateway errors with HTML responses
			//nolint:exhaustive
			switch urn {
			case codes.UnkeyGatewayErrorsRoutingConfigNotFound:
				return s.HTML(http.StatusNotFound, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>404 Not Found</title>
</head>
<body>
   <h1>404 Not Found</h1>
   <p>No gateway configuration found for this hostname.</p>
   <p>Please check your domain configuration.</p>
   <a href="/">Return to homepage</a>
</body>
</html>`))

			case codes.UnkeyGatewayErrorsProxyBadGateway,
				codes.UnkeyGatewayErrorsProxyProxyForwardFailed:
				return s.HTML(http.StatusBadGateway, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>502 Bad Gateway</title>
</head>
<body>
   <h1>502 Bad Gateway</h1>
   <p>The server received an invalid response from an upstream server.</p>
   <p>Please try again later.</p>
   <a href="/">Return to homepage</a>
</body>
</html>`))

			case codes.UnkeyGatewayErrorsProxyServiceUnavailable:
				return s.HTML(http.StatusServiceUnavailable, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>503 Service Unavailable</title>
</head>
<body>
   <h1>503 Service Unavailable</h1>
   <p>The service is temporarily unavailable.</p>
   <p>Please try again later.</p>
   <a href="/">Return to homepage</a>
</body>
</html>`))

			case codes.UnkeyGatewayErrorsProxyGatewayTimeout:
				return s.HTML(http.StatusGatewayTimeout, []byte(`<!DOCTYPE html>
<html lang="en">
<head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>504 Gateway Timeout</title>
</head>
<body>
   <h1>504 Gateway Timeout</h1>
   <p>The upstream server failed to respond within the timeout period.</p>
   <p>Please try again later.</p>
   <a href="/">Return to homepage</a>
</body>
</html>`))

			// All other errors (including App, Auth, Data errors) return JSON
			case codes.UserErrorsBadRequestPermissionsQuerySyntaxError,
				codes.UserErrorsBadRequestRequestBodyTooLarge,
				codes.UserErrorsBadRequestRequestTimeout,
				codes.UserErrorsBadRequestClientClosedRequest,
				codes.UnkeyAuthErrorsAuthenticationMissing,
				codes.UnkeyAuthErrorsAuthenticationMalformed,
				codes.UnkeyAuthErrorsAuthenticationKeyNotFound,
				codes.UnkeyAuthErrorsAuthorizationInsufficientPermissions,
				codes.UnkeyAuthErrorsAuthorizationForbidden,
				codes.UnkeyAuthErrorsAuthorizationKeyDisabled,
				codes.UnkeyAuthErrorsAuthorizationWorkspaceDisabled,
				codes.UnkeyDataErrorsKeyNotFound,
				codes.UnkeyDataErrorsWorkspaceNotFound,
				codes.UnkeyDataErrorsApiNotFound,
				codes.UnkeyDataErrorsPermissionDuplicate,
				codes.UnkeyDataErrorsPermissionNotFound,
				codes.UnkeyDataErrorsRoleDuplicate,
				codes.UnkeyDataErrorsRoleNotFound,
				codes.UnkeyDataErrorsKeyAuthNotFound,
				codes.UnkeyDataErrorsRatelimitNamespaceNotFound,
				codes.UnkeyDataErrorsRatelimitNamespaceGone,
				codes.UnkeyDataErrorsRatelimitOverrideNotFound,
				codes.UnkeyDataErrorsIdentityNotFound,
				codes.UnkeyDataErrorsIdentityDuplicate,
				codes.UnkeyDataErrorsAuditLogNotFound,
				codes.UnkeyAppErrorsInternalUnexpectedError,
				codes.UnkeyAppErrorsInternalServiceUnavailable,
				codes.UnkeyAppErrorsValidationInvalidInput,
				codes.UnkeyAppErrorsValidationAssertionFailed,
				codes.UnkeyAppErrorsProtectionProtectedResource,
				codes.UnkeyAppErrorsPreconditionPreconditionFailed,
				codes.UnkeyGatewayErrorsValidationRequestInvalid,
				codes.UnkeyGatewayErrorsValidationResponseInvalid,
				codes.UnkeyGatewayErrorsAuthUnauthorized,
				codes.UnkeyGatewayErrorsAuthRateLimited,
				codes.UnkeyGatewayErrorsInternalInternalServerError,
				codes.UnkeyGatewayErrorsInternalKeyVerificationFailed,
				codes.UnkeyGatewayErrorsRoutingVMSelectionFailed:
				// Return JSON for these errors
			}

			// Create error response
			return s.JSON(status, ErrorResponse{
				Meta: Meta{
					RequestID: s.RequestID(),
				},
				Error: ErrorDetail{
					Title:  http.StatusText(status),
					Type:   code.DocsURL(),
					Detail: fault.UserFacingMessage(err),
					Status: status,
				},
			})
		}
	}
}
