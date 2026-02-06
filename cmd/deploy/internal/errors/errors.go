package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

// appendMetadata adds request ID and documentation link to the error message
func appendMetadata(msg *strings.Builder, meta components.Meta, errorType string) {
	// Add request ID for debugging
	if requestID := meta.GetRequestID(); requestID != "" {
		fmt.Fprintf(msg, "\n\nRequest ID: %s", requestID)
	}

	// Add documentation link if available
	if errorType != "" {
		fmt.Fprintf(msg, "\nDocumentation: %s", errorType)
	}
}

// formatPermissionError creates a user-friendly error message for permission errors
func formatPermissionError(apiErr *apierrors.ForbiddenErrorResponse) string {
	var msg strings.Builder
	msg.WriteString("Permission denied\n\n")
	msg.WriteString(apiErr.Error_.GetDetail())
	msg.WriteString("\n\nTo fix this, update your root key permissions in the Unkey dashboard.")

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

// formatAuthenticationError creates a user-friendly error message for authentication errors
func formatAuthenticationError(apiErr *apierrors.UnauthorizedErrorResponse) string {
	var msg strings.Builder
	msg.WriteString("Authentication failed\n\n")

	// Provide specific guidance based on the error detail
	detail := strings.ToLower(apiErr.Error_.GetDetail())
	if strings.Contains(detail, "invalid") || strings.Contains(detail, "token") {
		msg.WriteString("Your root key is invalid or has expired.\n\n")
		msg.WriteString("Please check:\n")
		msg.WriteString("  • The root key is correct (--root-key flag or UNKEY_ROOT_KEY env var)\n")
		msg.WriteString("  • The key hasn't been revoked or deleted\n")
		msg.WriteString("  • You're using a root key, not an API key\n")
	} else {
		msg.WriteString("Could not authenticate your request.\n\n")
		msg.WriteString("Make sure you've provided a valid root key using:\n")
		msg.WriteString("  • --root-key flag, or\n")
		msg.WriteString("  • UNKEY_ROOT_KEY environment variable\n")
	}

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

// formatNotFoundError creates a user-friendly error message for not found errors
func formatNotFoundError(apiErr *apierrors.NotFoundErrorResponse) string {
	var msg strings.Builder

	// Determine what wasn't found based on the error details
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
		msg.WriteString(apiErr.Error_.GetDetail())
	}

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

// formatValidationError creates a clean error message for validation errors
func formatValidationError(apiErr *apierrors.BadRequestErrorResponse) string {
	var msg strings.Builder

	detail := apiErr.Error_.GetDetail()
	if detail != "" {
		msg.WriteString(detail)
		msg.WriteString("\n\n")
	}

	validationErrors := apiErr.Error_.GetErrors()
	if len(validationErrors) > 0 {
		msg.WriteString("Validation errors:\n")
		for _, verr := range validationErrors {
			location := verr.GetLocation()
			message := verr.GetMessage()
			msg.WriteString(fmt.Sprintf("  • %s: %s\n", location, message))
		}
	}

	appendMetadata(&msg, apiErr.Meta, apiErr.Error_.GetType())

	return msg.String()
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var forbiddenErr *apierrors.ForbiddenErrorResponse
	if errors.As(err, &forbiddenErr) {
		return formatPermissionError(forbiddenErr)
	}

	var unauthorizedErr *apierrors.UnauthorizedErrorResponse
	if errors.As(err, &unauthorizedErr) {
		return formatAuthenticationError(unauthorizedErr)
	}

	var notFoundErr *apierrors.NotFoundErrorResponse
	if errors.As(err, &notFoundErr) {
		return formatNotFoundError(notFoundErr)
	}

	var badRequestErr *apierrors.BadRequestErrorResponse
	if errors.As(err, &badRequestErr) {
		return formatValidationError(badRequestErr)
	}

	return err.Error()
}
