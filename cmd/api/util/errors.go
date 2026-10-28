package util

import (
	"errors"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
)

// FormatError converts SDK error types into human-readable messages.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	if forbidden, ok := errors.AsType[*apierrors.ForbiddenErrorResponse](err); ok {
		return errorDiagnostic("Permission denied: "+terminalText(forbidden.Error_.GetDetail()), forbidden.Meta.RequestID)
	}

	if unauthorized, ok := errors.AsType[*apierrors.UnauthorizedErrorResponse](err); ok {
		return errorDiagnostic(fmt.Sprintf("Authentication failed: %s\n\nCheck your root key or run 'unkey login'", terminalText(unauthorized.Error_.GetDetail())), unauthorized.Meta.RequestID)
	}

	if notFound, ok := errors.AsType[*apierrors.NotFoundErrorResponse](err); ok {
		return errorDiagnostic("Not found: "+terminalText(notFound.Error_.GetDetail()), notFound.Meta.RequestID)
	}

	if badRequest, ok := errors.AsType[*apierrors.BadRequestErrorResponse](err); ok {
		msg := terminalText(badRequest.Error_.GetDetail())
		for _, ve := range badRequest.Error_.GetErrors() {
			msg += fmt.Sprintf("\n  %s: %s", terminalText(ve.GetLocation()), terminalText(ve.GetMessage()))
		}
		return errorDiagnostic(msg, badRequest.Meta.RequestID)
	}

	if server, ok := errors.AsType[*apierrors.InternalServerErrorResponse](err); ok {
		return errorDiagnostic("Server error: "+terminalText(server.Error_.Detail), server.Meta.RequestID)
	}
	if unavailable, ok := errors.AsType[*apierrors.ServiceUnavailableErrorResponse](err); ok {
		return errorDiagnostic("Service unavailable: "+terminalText(unavailable.Error_.Detail), unavailable.Meta.RequestID)
	}
	if limited, ok := errors.AsType[*apierrors.TooManyRequestsErrorResponse](err); ok {
		return errorDiagnostic("Rate limited: "+terminalText(limited.Error_.Detail), limited.Meta.RequestID)
	}
	if conflict, ok := errors.AsType[*apierrors.ConflictErrorResponse](err); ok {
		return errorDiagnostic("Conflict: "+terminalText(conflict.Error_.Detail), conflict.Meta.RequestID)
	}
	if gone, ok := errors.AsType[*apierrors.GoneErrorResponse](err); ok {
		return errorDiagnostic(terminalText(gone.Error_.Detail), gone.Meta.RequestID)
	}
	if precondition, ok := errors.AsType[*apierrors.PreconditionFailedErrorResponse](err); ok {
		return errorDiagnostic(terminalText(precondition.Error_.Detail), precondition.Meta.RequestID)
	}
	if invalid, ok := errors.AsType[*apierrors.UnprocessableEntityErrorResponse](err); ok {
		return errorDiagnostic(terminalText(invalid.Error_.Detail), invalid.Meta.RequestID)
	}
	if response, ok := errors.AsType[*apierrors.APIError](err); ok {
		return fmt.Sprintf("API request failed (HTTP %d)", response.StatusCode)
	}
	return err.Error()
}

func errorDiagnostic(message, requestID string) string {
	if requestID == "" {
		return message
	}
	return message + "\nRequest ID: " + terminalText(requestID)
}
