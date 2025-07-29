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

			// Bad Request errors
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
			case codes.UnkeyDataErrorsIdentityDuplicate,
				codes.UnkeyDataErrorsRoleDuplicate,
				codes.UnkeyDataErrorsPermissionDuplicate:
				return s.JSON(http.StatusConflict, openapi.ConflictErrorResponse{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Error: openapi.BaseError{
						Title:  "Duplicate Identity",
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
			case codes.UnkeyAppErrorsPreconditionPreconditionFailed:
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
