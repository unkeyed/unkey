package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// errorLogAttrs builds the structured attributes attached to every 5xx log
// from this middleware. The goal is to give an on-call engineer enough
// context to find the failing request in ClickHouse / wide-event logs
// without having to grep around: workspace and request id pin the row,
// http.* describes what was called, error.* (added by the logger's fault
// handler) describes what went wrong.
//
// We deliberately don't include the request body — it can contain secrets
// and is already redacted-and-logged by the wide-event request logger.
// Key id / identity id are not available here (they live on KeyVerifier,
// not zen.Session), but the request id lets you pivot to the wide event
// that does carry them.
func errorLogAttrs(s *zen.Session, err error, status int, urn codes.URN) []any {
	workspaceID := ""
	if principal, principalErr := s.GetPrincipal(); principalErr == nil {
		workspaceID = principal.WorkspaceID
	}

	return []any{
		"error", err,
		"workspaceId", workspaceID,
		"requestId", s.RequestID(),
		"code", string(urn),
		"publicMessage", fault.UserFacingMessage(err),
		slog.Group("http",
			slog.String("method", s.Request().Method),
			slog.String("path", s.Request().URL.Path),
			slog.String("query", s.Request().URL.RawQuery),
			slog.String("host", s.Request().Host),
			slog.String("user_agent", s.UserAgent()),
			slog.String("ip", s.Location()),
			slog.String("referer", s.Request().Referer()),
			slog.Int("status", status),
		),
	}
}

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
				codes.UnkeyDataErrorsAppNotFound,
				codes.UnkeyDataErrorsMigrationNotFound,
				codes.UnkeyDataErrorsKeySpaceNotFound,
				codes.UnkeyDataErrorsPermissionNotFound,
				codes.UnkeyDataErrorsProjectNotFound,
				codes.UnkeyDataErrorsRoleNotFound,
				codes.UnkeyDataErrorsKeyAuthNotFound,
				codes.UnkeyDataErrorsRatelimitNamespaceNotFound,
				codes.UnkeyDataErrorsRatelimitOverrideNotFound,
				codes.UnkeyDataErrorsIdentityNotFound,
				codes.UnkeyDataErrorsAuditLogNotFound,
				codes.UnkeyDataErrorsPortalConfigNotFound:
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
				codes.UserErrorsBadRequestRequestBodyUnreadable,
				codes.UnkeyPortalErrorsSessionTokenMissing:
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

			// Too Many Requests - Rate limiting
			case codes.UserErrorsTooManyRequestsQueryQuotaExceeded,
				codes.UserErrorsTooManyRequestsWorkspaceRateLimited:
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
			case codes.UnkeyAuthErrorsAuthenticationKeyNotFound,
				codes.UnkeyPortalErrorsSessionSessionNotFound:
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
				codes.UnkeyDataErrorsPermissionDuplicate,
				codes.UnkeyDataErrorsProjectDuplicate,
				codes.UnkeyDataErrorsAppDuplicate:
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
					errorLogAttrs(s, err, http.StatusServiceUnavailable, urn)...,
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
				errorLogAttrs(s, err, http.StatusInternalServerError, urn)...,
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
