package util

import (
	"errors"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/apierrors"
)

// FormatError converts SDK error types into human-readable messages.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var forbidden *apierrors.ForbiddenErrorResponse
	if errors.As(err, &forbidden) {
		return fmt.Sprintf("Permission denied: %s", forbidden.Error_.GetDetail())
	}

	var unauthorized *apierrors.UnauthorizedErrorResponse
	if errors.As(err, &unauthorized) {
		return fmt.Sprintf("Authentication failed: %s\n\nCheck your root key or run 'unkey auth login'", unauthorized.Error_.GetDetail())
	}

	var notFound *apierrors.NotFoundErrorResponse
	if errors.As(err, &notFound) {
		return fmt.Sprintf("Not found: %s", notFound.Error_.GetDetail())
	}

	var badRequest *apierrors.BadRequestErrorResponse
	if errors.As(err, &badRequest) {
		msg := badRequest.Error_.GetDetail()
		for _, ve := range badRequest.Error_.GetErrors() {
			msg += fmt.Sprintf("\n  %s: %s", ve.GetLocation(), ve.GetMessage())
		}
		return msg
	}

	return err.Error()
}
