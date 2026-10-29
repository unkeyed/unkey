package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"syscall"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"go.opentelemetry.io/otel/trace"
)

func logRequestError(ctx context.Context, s *zen.Session, err, responseErr error) {
	status := s.StatusCode()
	if err == nil && responseErr == nil && status < http.StatusInternalServerError {
		return
	}

	urn, _ := fault.GetCode(err)
	category, _ := fault.GetCategory(err)
	clientCanceled := errors.Is(s.Request().Context().Err(), context.Canceled)
	writeFailed := responseErr != nil && !(clientCanceled &&
		(errors.Is(responseErr, context.Canceled) || errors.Is(responseErr, syscall.EPIPE) || errors.Is(responseErr, syscall.ECONNRESET)))
	operational := status >= http.StatusInternalServerError || writeFailed || category == codes.CategoryPlatform ||
		urn == codes.User.BadRequest.RequestTimeout.URN() ||
		(urn == codes.User.BadRequest.ClientClosedRequest.URN() && !(clientCanceled && errors.Is(err, context.Canceled)))

	var authorizationErr error
	p, principalErr := s.GetPrincipal()
	if principalErr == nil {
		authorizationErr = principal.AuthorizationError(p)
	}
	code, parseErr := codes.ParseURN(urn)
	security := status >= http.StatusBadRequest && (authorizationErr != nil || (parseErr == nil &&
		(code.Category == codes.CategoryUnkeyAuthentication || code.Category == codes.CategoryUnkeyAuthorization)))
	if !operational && !security {
		return
	}
	if err == nil {
		err = fault.New("API returned a server error", fault.Internal(s.InternalError()))
	}
	if urn == "" {
		urn = codes.App.Internal.UnexpectedError.URN()
	}

	attrs := []any{
		"error", err,
		"requestId", s.RequestID(),
		"code", string(urn),
		"faultDomain", string(category),
		"responseSource", "api",
		"publicMessage", fault.UserFacingMessage(err),
		slog.Group("http",
			slog.String("method", s.Request().Method),
			slog.String("path", s.Request().URL.Path),
			slog.String("host", s.Request().Host),
			slog.String("user_agent", s.UserAgent()),
			slog.String("ip", s.Location()),
			slog.Int("status", status),
		),
	}
	if principalErr == nil {
		attrs = append(attrs,
			"workspaceId", p.AuthorizedWorkspaceID,
			"subjectId", p.Subject.ID,
			"subjectType", string(p.Subject.Type),
			"authType", string(p.Type),
		)
		if authorizationErr != nil {
			attrs = append(attrs, "authorizationError", authorizationErr, "grantedPermissions", p.Permissions)
		}
	}
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		attrs = append(attrs, "traceId", spanContext.TraceID().String(), "spanId", spanContext.SpanID().String())
	}
	if writeFailed {
		attrs = append(attrs, "responseError", responseErr)
	}
	if operational {
		slog.ErrorContext(ctx, "api error", append(attrs, "event", "api.request_error")...)
	} else {
		slog.InfoContext(ctx, "api security denial", append(attrs, "event", "api.security_denial")...)
	}
}

// WithErrorHandling translates application and routing errors into HTTP responses.
func WithErrorHandling() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) (responseErr error) {
			err := next(ctx, s)
			defer func() { logRequestError(ctx, s, err, responseErr) }()
			if err == nil {
				return nil
			}

			// Store the internal error message for metrics logging before we
			// convert it to an HTTP response and lose the details.
			s.SetInternalError(fault.InternalMessage(err))

			if routingErr, ok := errors.AsType[*zen.RoutingError](err); ok {
				detail := "The requested endpoint does not exist."
				if routingErr.Status == http.StatusMethodNotAllowed {
					detail = "The request method is not supported for this endpoint."
				}
				return s.ProblemJSON(routingErr.Status, struct {
					Meta  openapi.Meta      `json:"meta"`
					Error openapi.BaseError `json:"error"`
				}{
					Meta: openapi.Meta{RequestId: s.RequestID()},
					Error: openapi.BaseError{
						Title:  http.StatusText(routingErr.Status),
						Type:   "about:blank",
						Detail: detail,
						Status: routingErr.Status,
					},
				})
			}

			// Get the error URN from the error
			urn, ok := fault.GetCode(err)
			if !ok {
				urn = codes.App.Internal.UnexpectedError.URN()
			}

			code, parseErr := codes.ParseURN(urn)
			if parseErr != nil {
				code = codes.App.Internal.UnexpectedError
			}
			//nolint:exhaustive
			switch urn {
			// Not Found errors
			case codes.UnkeyDataErrorsKeyNotFound,
				codes.UnkeyDataErrorsFlagNotFound,
				codes.UnkeyDataErrorsWorkspaceNotFound,
				codes.UnkeyDataErrorsApiNotFound,
				codes.UnkeyDataErrorsAppNotFound,
				codes.UnkeyDataErrorsEnvironmentNotFound,
				codes.UnkeyDataErrorsDomainNotFound,
				codes.UnkeyDataErrorsDeploymentNotFound,
				codes.UnkeyDataErrorsPolicyNotFound,
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
				codes.UnkeyDataErrorsPortalNotFound:
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
				codes.UnkeyAppErrorsValidationInvalidEnvironmentSettings,
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
				codes.UserErrorsBadRequestQueryRangeExceedsRetention,
				codes.UserErrorsBadRequestPerKeyBreakoutTooLarge:
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
				codes.UserErrorsTooManyRequestsWorkspaceRateLimited,
				codes.UserErrorsTooManyRequestsIPRateLimited,
				codes.UserErrorsTooManyRequestsUserRateLimited:
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
			case codes.UnkeyDataErrorsRatelimitNamespaceGone,
				codes.UnkeyAuthErrorsAuthenticationDeviceLoginExpired:
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
				codes.UnkeyAuthErrorsAuthenticationCredentialRejected,
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
						Detail: RedactResourceIDs(fault.UserFacingMessage(err)),
						Status: http.StatusForbidden,
					},
				})

			// Plan allowance exhausted. Forbidden rather than 429: no amount of
			// waiting changes the answer, the caller has to change their plan or
			// free up an existing resource.
			case codes.UnkeyLimitsErrorsCustomDomainExceeded:
				return s.ProblemJSON(http.StatusForbidden, openapi.ForbiddenErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Forbidden",
						Type:   code.DocsURL(),
						Detail: RedactResourceIDs(fault.UserFacingMessage(err)),
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
						Detail: RedactResourceIDs(fault.UserFacingMessage(err)),
						Status: http.StatusForbidden,
					},
				})

			// Duplicate errors
			case codes.UnkeyDataErrorsIdentityDuplicate,
				codes.UnkeyDataErrorsRoleDuplicate,
				codes.UnkeyDataErrorsPermissionDuplicate,
				codes.UnkeyDataErrorsProjectDuplicate,
				codes.UnkeyDataErrorsAppDuplicate,
				codes.UnkeyDataErrorsDomainDuplicate,
				codes.UnkeyDataErrorsPortalDuplicate,
				codes.UnkeyDataErrorsPortalChanged,
				codes.UnkeyDataErrorsRatelimitNamespaceDuplicate:
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
				codes.UnkeyAppErrorsPreconditionPreconditionFailed,
				codes.UnkeyAppErrorsPreconditionDeploymentNotReady,
				codes.UnkeyAppErrorsPreconditionDeploymentNotProduction,
				codes.UnkeyAppErrorsPreconditionDeploymentNoCurrent,
				codes.UnkeyAppErrorsPreconditionDeploymentIsCurrent,
				codes.UnkeyAppErrorsPreconditionDeploymentNotRunning,
				codes.UnkeyAppErrorsPreconditionDeploymentIsStopping,
				codes.UnkeyAppErrorsPreconditionDeploymentNotStopped,
				codes.UnkeyAppErrorsPreconditionDeploymentIsProduction,
				codes.UnkeyAppErrorsPreconditionDeploymentNotAwaitingApproval,
				codes.UnkeyAppErrorsPreconditionDeploymentNotInProgress:
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
						Detail: RedactResourceIDs(fault.UserFacingMessage(err)),
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
						Detail: RedactResourceIDs(fault.UserFacingMessage(err)),
						Status: http.StatusForbidden,
					},
				})

			// Service Unavailable errors
			case codes.UnkeyDataErrorsAnalyticsConnectionFailed:
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
