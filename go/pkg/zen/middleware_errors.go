package zen

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

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

			switch urn {
			// Not Found errors
			case codes.UnkeyDataErrorsKeyNotFound,
				codes.UnkeyDataErrorsWorkspaceNotFound,
				codes.UnkeyDataErrorsApiNotFound,
				codes.UnkeyDataErrorsKeySpaceNotFound,
				codes.UnkeyDataErrorsPermissionNotFound,
				codes.UnkeyDataErrorsRoleNotFound,
				codes.UnkeyDataErrorsKeyAuthNotFound,
				codes.UnkeyDataErrorsRatelimitNamespaceNotFound,
				codes.UnkeyDataErrorsRatelimitOverrideNotFound,
				codes.UnkeyDataErrorsIdentityNotFound,
				codes.UnkeyDataErrorsAuditLogNotFound:
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

			// Bad Request errors - General validation
			case codes.UnkeyAppErrorsValidationInvalidInput,
				codes.UnkeyAuthErrorsAuthenticationMissing,
				codes.UnkeyAuthErrorsAuthenticationMalformed,
				codes.UserErrorsBadRequestPermissionsQuerySyntaxError:
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

			// Bad Request errors - Query validation (malformed queries)
			case codes.UserErrorsBadRequestInvalidAnalyticsQuery,
				codes.UserErrorsBadRequestInvalidAnalyticsTable,
				codes.UserErrorsBadRequestInvalidAnalyticsFunction,
				codes.UserErrorsBadRequestInvalidAnalyticsQueryType:
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

			// Unprocessable Entity - Query resource limits
			case codes.UserErrorsUnprocessableEntityQueryExecutionTimeout,
				codes.UserErrorsUnprocessableEntityQueryMemoryLimitExceeded,
				codes.UserErrorsUnprocessableEntityQueryRowsLimitExceeded:
				return s.JSON(http.StatusUnprocessableEntity, openapi.UnprocessableEntityErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  http.StatusText(http.StatusUnprocessableEntity),
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusUnprocessableEntity,
					},
				})

			// Too Many Requests - Query rate limiting
			case codes.UserErrorsTooManyRequestsQueryQuotaExceeded:
				return s.JSON(http.StatusTooManyRequests, openapi.TooManyRequestsErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  http.StatusText(http.StatusTooManyRequests),
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusTooManyRequests,
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

			// Gone errors
			case codes.UnkeyDataErrorsRatelimitNamespaceGone:
				return s.JSON(http.StatusGone, openapi.GoneErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Resource Gone",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusGone,
					},
				})

			// Unauthorized errors
			case
				codes.UnkeyAuthErrorsAuthenticationKeyNotFound:
				return s.JSON(http.StatusUnauthorized, openapi.UnauthorizedErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Unauthorized",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusUnauthorized,
					},
				})

			// Forbidden errors
			case codes.UnkeyAuthErrorsAuthorizationForbidden:
				return s.JSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Forbidden",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusForbidden,
					},
				})

			// Insufficient Permissions
			case codes.UnkeyAuthErrorsAuthorizationInsufficientPermissions:
				return s.JSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Insufficient Permissions",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusForbidden,
					},
				})

			// Duplicate errors
			case codes.UnkeyDataErrorsIdentityDuplicate,
				codes.UnkeyDataErrorsRoleDuplicate,
				codes.UnkeyDataErrorsPermissionDuplicate:
				return s.JSON(http.StatusConflict, openapi.ConflictErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Conflicting Resource",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusConflict,
					},
				})

			// Protected Resource
			case codes.UnkeyAppErrorsProtectionProtectedResource:
				return s.JSON(http.StatusPreconditionFailed, openapi.PreconditionFailedErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Resource is protected",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusPreconditionFailed,
					},
				})

			// Precondition Failed
			case codes.UnkeyDataErrorsAnalyticsNotConfigured,
				codes.UnkeyAppErrorsPreconditionPreconditionFailed:
				return s.JSON(http.StatusPreconditionFailed, openapi.PreconditionFailedErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Precondition Failed",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusPreconditionFailed,
					},
				})

			// Key disabled
			case codes.UnkeyAuthErrorsAuthorizationKeyDisabled:
				return s.JSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Key is disabled",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusForbidden,
					},
				})

			// Workspace disabled
			case codes.UnkeyAuthErrorsAuthorizationWorkspaceDisabled:
				return s.JSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Workspace is disabled",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusForbidden,
					},
				})

			// Service Unavailable errors
			case codes.UnkeyDataErrorsAnalyticsConnectionFailed:
				return s.JSON(http.StatusServiceUnavailable, openapi.InternalServerErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Service Unavailable",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusServiceUnavailable,
					},
				})

			// Internal errors
			case codes.UnkeyAppErrorsInternalUnexpectedError,
				codes.UnkeyAppErrorsInternalServiceUnavailable,
				codes.UnkeyAppErrorsValidationAssertionFailed:
				// Fall through to default 500 error
			}

			logger.Error("api error",
				"error", err.Error(),
				"requestId", s.RequestID(),
				"publicMessage", fault.UserFacingMessage(err),
			)

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
