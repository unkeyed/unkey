package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/deploy/internal/ui"
)

func appendMetadata(msg *strings.Builder, meta components.Meta, errorType string) {
	if requestID := meta.GetRequestID(); requestID != "" {
		fmt.Fprintf(msg, "\n\nRequest ID: %s", ui.Text(requestID))
	}

	if errorType != "" {
		fmt.Fprintf(msg, "\nDocumentation: %s", ui.Text(errorType))
	}
}

func formatPermissionError(apiErr *apierrors.ForbiddenErrorResponse) string {
	var msg strings.Builder
	msg.WriteString("Permission denied\n\n")
	msg.WriteString(ui.Text(apiErr.Error_.GetDetail()))
	msg.WriteString("\n\nTo fix this, update your root key permissions in the Unkey dashboard.")

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

func formatAuthenticationError(apiErr *apierrors.UnauthorizedErrorResponse) string {
	var msg strings.Builder
	msg.WriteString("Authentication failed\n\n")

	detail := strings.ToLower(apiErr.Error_.GetDetail())
	if strings.Contains(detail, "invalid") || strings.Contains(detail, "token") {
		msg.WriteString("The credential for unkey deploy is invalid or has expired.\n\n")
		msg.WriteString("Please check:\n")
		msg.WriteString("  • --root-key, when you passed one\n")
		msg.WriteString("  • UNKEY_ROOT_KEY\n")
		msg.WriteString("  • the CLI session from unkey login, for the API that issued it\n")
		msg.WriteString("  • a root key saved with unkey login --root-key\n")
	} else {
		msg.WriteString("Could not authenticate unkey deploy.\n\n")
		msg.WriteString("Provide a credential in this order:\n")
		msg.WriteString("  • --root-key\n")
		msg.WriteString("  • UNKEY_ROOT_KEY\n")
		msg.WriteString("  • unkey login\n")
		msg.WriteString("  • unkey login --root-key\n")
	}

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

func formatNotFoundError(apiErr *apierrors.NotFoundErrorResponse) string {
	var msg strings.Builder

	detail := strings.ToLower(apiErr.Error_.GetDetail())
	errorType := strings.ToLower(apiErr.Error_.GetType())

	if strings.Contains(detail, "deployment") {
		msg.WriteString("Deployment not found\n\n")
		msg.WriteString("The specified deployment doesn't exist or you don't have access to it.")
	} else if strings.Contains(detail, "project") || strings.Contains(errorType, "project") {
		msg.WriteString("Project not found\n\n")
		msg.WriteString("The specified project doesn't exist or you don't have access to it.")
	} else if strings.Contains(detail, "environment") {
		msg.WriteString("Environment not found\n\n")
		msg.WriteString("The specified environment doesn't exist in this project.")
	} else {
		msg.WriteString("Resource not found\n\n")
		msg.WriteString(ui.Text(apiErr.Error_.GetDetail()))
	}

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

func formatValidationError(apiErr *apierrors.BadRequestErrorResponse) string {
	var msg strings.Builder

	detail := apiErr.Error_.GetDetail()
	if detail != "" {
		msg.WriteString(ui.Text(detail))
		msg.WriteString("\n\n")
	}

	validationErrors := apiErr.Error_.GetErrors()
	if len(validationErrors) > 0 {
		msg.WriteString("Validation errors:\n")
		for _, verr := range validationErrors {
			location := verr.GetLocation()
			message := verr.GetMessage()
			msg.WriteString(fmt.Sprintf("  • %s: %s\n", ui.Text(location), ui.Text(message)))
		}
	}

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}

	if forbiddenErr, ok := errors.AsType[*apierrors.ForbiddenErrorResponse](err); ok {
		return formatPermissionError(forbiddenErr)
	}

	if unauthorizedErr, ok := errors.AsType[*apierrors.UnauthorizedErrorResponse](err); ok {
		return formatAuthenticationError(unauthorizedErr)
	}

	if notFoundErr, ok := errors.AsType[*apierrors.NotFoundErrorResponse](err); ok {
		return formatNotFoundError(notFoundErr)
	}

	if badRequestErr, ok := errors.AsType[*apierrors.BadRequestErrorResponse](err); ok {
		return formatValidationError(badRequestErr)
	}

	return ui.Text(err.Error())
}
