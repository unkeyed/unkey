package middleware

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// WithErrorHandling returns middleware that translates errors into appropriate
// HTTP responses based on error URNs.
func WithErrorHandling() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
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
			//nolint:exhaustive
			switch urn {
			// Not Found errors
			case codes.UnkeyDataErrorsKeyNotFound,
				codes.UnkeyDataErrorsWorkspaceNotFound,
				codes.UnkeyDataErrorsApiNotFound,
				codes.UnkeyDataErrorsMigrationNotFound,
				codes.UnkeyDataErrorsKeySpaceNotFound,
				codes.UnkeyDataErrorsPermissionNotFound,
				codes.UnkeyDataErrorsProjectNotFound,
				codes.UnkeyDataErrorsRoleNotFound,
				codes.UnkeyDataErrorsKeyAuthNotFound,
				codes.UnkeyDataErrorsRatelimitNamespaceNotFound,
				codes.UnkeyDataErrorsRatelimitOverrideNotFound,
				codes.UnkeyDataErrorsIdentityNotFound,
				codes.UnkeyDataErrorsAuditLogNotFound:
				return s.ProblemJSON(http.StatusNotFound, openapi.NotFoundErrorResponse{
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
				codes.UserErrorsBadRequestPermissionsQuerySyntaxError,
				codes.UserErrorsBadRequestRequestBodyUnreadable:
				return s.ProblemJSON(http.StatusBadRequest, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Bad Request",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusBadRequest,
						Errors: []openapi.ValidationError{},
						Schema: nil,
					},
				})

			// Bad Request errors - Query validation (malformed queries)
			case codes.UserErrorsBadRequestInvalidAnalyticsQuery,
				codes.UserErrorsBadRequestInvalidAnalyticsTable,
				codes.UserErrorsBadRequestInvalidAnalyticsFunction,
				codes.UserErrorsBadRequestInvalidAnalyticsQueryType,
				codes.UserErrorsBadRequestQueryRangeExceedsRetention:
				return s.ProblemJSON(http.StatusBadRequest, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Bad Request",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusBadRequest,
						Errors: []openapi.ValidationError{},
						Schema: nil,
					},
				})

			// Unprocessable Entity - Query resource limits
			case codes.UserErrorsUnprocessableEntityQueryExecutionTimeout,
				codes.UserErrorsUnprocessableEntityQueryMemoryLimitExceeded,
				codes.UserErrorsUnprocessableEntityQueryRowsLimitExceeded:
				return s.ProblemJSON(http.StatusUnprocessableEntity, openapi.UnprocessableEntityErrorResponse{
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
				return s.ProblemJSON(http.StatusTooManyRequests, openapi.TooManyRequestsErrorResponse{
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
				return s.ProblemJSON(http.StatusRequestTimeout, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Request Timeout",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusRequestTimeout,
						Errors: []openapi.ValidationError{},
						Schema: nil,
					},
				})

			// Client Closed Request errors (499 - non-standard but widely used)
			case codes.UserErrorsBadRequestClientClosedRequest:
				return s.ProblemJSON(499, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Client Closed Request",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: 499,
						Errors: []openapi.ValidationError{},
						Schema: nil,
					},
				})

			// Request Entity Too Large errors
			case codes.UserErrorsBadRequestRequestBodyTooLarge:
				return s.ProblemJSON(http.StatusRequestEntityTooLarge, openapi.BadRequestErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BadRequestErrorDetails{
						Title:  "Request Entity Too Large",
						Type:   code.DocsURL(),
						Detail: fault.UserFacingMessage(err),
						Status: http.StatusRequestEntityTooLarge,
						Errors: []openapi.ValidationError{},
						Schema: nil,
					},
				})

			// Gone errors
			case codes.UnkeyDataErrorsRatelimitNamespaceGone:
				return s.ProblemJSON(http.StatusGone, openapi.GoneErrorResponse{
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
			case codes.UnkeyAuthErrorsAuthenticationKeyNotFound:
				return s.ProblemJSON(http.StatusUnauthorized, openapi.UnauthorizedErrorResponse{
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
				return s.ProblemJSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
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
				return s.ProblemJSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
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
				return s.ProblemJSON(http.StatusConflict, openapi.ConflictErrorResponse{
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
				return s.ProblemJSON(http.StatusPreconditionFailed, openapi.PreconditionFailedErrorResponse{
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
				return s.ProblemJSON(http.StatusPreconditionFailed, openapi.PreconditionFailedErrorResponse{
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
				return s.ProblemJSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
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
				return s.ProblemJSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
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
				logger.Error(
					"analytics connection error",
					"error", err.Error(),
					"requestId", s.RequestID(),
					"publicMessage", fault.UserFacingMessage(err),
				)

				return s.ProblemJSON(http.StatusServiceUnavailable, openapi.ServiceUnavailableErrorResponse{
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

			return s.ProblemJSON(http.StatusInternalServerError, openapi.InternalServerErrorResponse{
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
