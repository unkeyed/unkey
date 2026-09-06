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
		return fmt.Sprintf("Permission denied: %s", forbidden.Error_.GetDetail())
	}

	if unauthorized, ok := errors.AsType[*apierrors.UnauthorizedErrorResponse](err); ok {
		return fmt.Sprintf("Authentication failed: %s\n\nCheck your root key or run 'unkey auth login'", unauthorized.Error_.GetDetail())
	}

	if notFound, ok := errors.AsType[*apierrors.NotFoundErrorResponse](err); ok {
		return fmt.Sprintf("Not found: %s", notFound.Error_.GetDetail())
	}

	if badRequest, ok := errors.AsType[*apierrors.BadRequestErrorResponse](err); ok {
		msg := badRequest.Error_.GetDetail()
		for _, ve := range badRequest.Error_.GetErrors() {
			msg += fmt.Sprintf("\n  %s: %s", ve.GetLocation(), ve.GetMessage())
		}
		return msg
	}

	return err.Error()
}
