package errors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

// APIErrorResponse represents the structured error response from the API
// This matches the SDK's error response format
type APIErrorResponse struct {
	Meta  components.Meta                   `json:"meta"`
	Error components.BadRequestErrorDetails `json:"error"`
}

// BaseAPIErrorResponse is for non-400 errors that don't have validation details
type BaseAPIErrorResponse struct {
	Meta  components.Meta      `json:"meta"`
	Error components.BaseError `json:"error"`
}

// ParsedError represents a parsed API error with either validation details or base error
type ParsedError struct {
	Meta             components.Meta
	Detail           string
	Status           int64
	Title            string
	Type             string
	ValidationErrors []components.ValidationError
}

// parseAPIError attempts to extract structured error information from SDK errors
// Returns nil if the error is not a structured API error
func parseAPIError(err error) *ParsedError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Try parsing as BadRequestErrorResponse
	var badReqErr APIErrorResponse
	if jsonErr := json.Unmarshal([]byte(errMsg), &badReqErr); jsonErr == nil {
		// Check if it actually has the error field populated
		if badReqErr.Error.Detail != "" || badReqErr.Error.Title != "" {
			return &ParsedError{
				Meta:             badReqErr.Meta,
				Detail:           badReqErr.Error.GetDetail(),
				Status:           badReqErr.Error.GetStatus(),
				Title:            badReqErr.Error.GetTitle(),
				Type:             badReqErr.Error.GetType(),
				ValidationErrors: badReqErr.Error.GetErrors(),
			}
		}
	}

	// Try parsing as BaseAPIErrorResponse
	var baseErr BaseAPIErrorResponse
	if jsonErr := json.Unmarshal([]byte(errMsg), &baseErr); jsonErr == nil {
		if baseErr.Error.Detail != "" || baseErr.Error.Title != "" {
			return &ParsedError{
				ValidationErrors: nil,
				Meta:             baseErr.Meta,
				Detail:           baseErr.Error.GetDetail(),
				Status:           baseErr.Error.GetStatus(),
				Title:            baseErr.Error.GetTitle(),
				Type:             baseErr.Error.GetType(),
			}
		}
	}

	return nil
}

// isPermissionError checks if the error is related to insufficient permissions
func isPermissionError(apiErr *ParsedError) bool {
	return apiErr.Status == 403
}

// isAuthenticationError checks if the error is related to authentication
func isAuthenticationError(apiErr *ParsedError) bool {
	return apiErr.Status == 401
}

// isNotFoundError checks if the error is a 404 not found error
func isNotFoundError(apiErr *ParsedError) bool {
	return apiErr.Status == 404
}

// isValidationError checks if the error has validation errors
func isValidationError(apiErr *ParsedError) bool {
	return len(apiErr.ValidationErrors) > 0
}

// parsePermissionError extracts and cleans permission strings from the error detail
// Input: "Missing one of these permissions: ['{ project.*.generate_upload_url []}', ...]"
// Output: ["project.*.generate_upload_url", ...]
func parsePermissionError(detail string) []string {
	// Pattern to match permission strings in the format: '{ permission.name []}'
	// We want to extract just the permission.name part
	re := regexp.MustCompile(`\{\s*([^\s\[\]{}]+)\s*\[\]\s*\}`)
	matches := re.FindAllStringSubmatch(detail, -1)

	var permissions []string
	seen := make(map[string]bool) // Deduplicate permissions

	for _, match := range matches {
		if len(match) > 1 {
			perm := strings.TrimSpace(match[1])
			if perm != "" && !seen[perm] {
				permissions = append(permissions, perm)
				seen[perm] = true
			}
		}
	}

	return permissions
}

// appendMetadata adds request ID and documentation link to the error message
func appendMetadata(msg *strings.Builder, apiErr *ParsedError) {
	// Add request ID for debugging
	if requestID := apiErr.Meta.GetRequestID(); requestID != "" {
		fmt.Fprintf(msg, "\n\nRequest ID: %s", requestID)
	}

	// Add documentation link if available
	if apiErr.Type != "" {
		fmt.Fprintf(msg, "\nDocumentation: %s", apiErr.Type)
	}
}

// formatPermissionError creates a user-friendly error message for permission errors
func formatPermissionError(apiErr *ParsedError) string {
	permissions := parsePermissionError(apiErr.Detail)

	var msg strings.Builder
	msg.WriteString("Missing required permissions\n\n")

	if len(permissions) == 0 {
		// Fallback if we can't parse permissions
		msg.WriteString(apiErr.Detail)
	} else if len(permissions) == 1 {
		msg.WriteString("You need this permission:\n")
		msg.WriteString(fmt.Sprintf("  • %s\n", permissions[0]))
	} else {
		msg.WriteString("You need ONE of the following permissions:\n")
		for _, perm := range permissions {
			msg.WriteString(fmt.Sprintf("  • %s\n", perm))
		}
	}

	if len(permissions) > 0 {
		msg.WriteString("\nTo fix this, add one of these permissions to your root key in the Unkey dashboard.")
	}

	appendMetadata(&msg, apiErr)

	return msg.String()
}

// formatAuthenticationError creates a user-friendly error message for authentication errors
func formatAuthenticationError(apiErr *ParsedError) string {
	var msg strings.Builder
	msg.WriteString("Authentication failed\n\n")

	// Provide specific guidance based on the error detail
	detail := strings.ToLower(apiErr.Detail)
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

	appendMetadata(&msg, apiErr)

	return msg.String()
}

// formatNotFoundError creates a user-friendly error message for not found errors
func formatNotFoundError(apiErr *ParsedError) string {
	var msg strings.Builder

	// Determine what wasn't found based on the error details
	detail := strings.ToLower(apiErr.Detail)
	errorType := strings.ToLower(apiErr.Type)

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
		msg.WriteString(apiErr.Detail)
	}

	appendMetadata(&msg, apiErr)

	return msg.String()
}

// formatValidationError creates a clean error message for validation errors
func formatValidationError(apiErr *ParsedError) string {
	var msg strings.Builder

	if apiErr.Detail != "" {
		msg.WriteString(apiErr.Detail)
		msg.WriteString("\n\n")
	}

	if len(apiErr.ValidationErrors) > 0 {
		msg.WriteString("Validation errors:\n")
		for _, verr := range apiErr.ValidationErrors {
			location := verr.GetLocation()
			message := verr.GetMessage()
			msg.WriteString(fmt.Sprintf("  • %s: %s\n", location, message))
		}
	}

	appendMetadata(&msg, apiErr)

	return msg.String()
}

// ErrorFormatter attempts to format a specific error type
// Returns (formatted string, true) if handled, ("", false) otherwise
type ErrorFormatter func(*ParsedError) (string, bool)

// tryFormatPermission handles permission errors
func tryFormatPermission(apiErr *ParsedError) (string, bool) {
	if !isPermissionError(apiErr) {
		return "", false
	}
	return formatPermissionError(apiErr), true
}

// tryFormatAuth handles authentication errors
func tryFormatAuth(apiErr *ParsedError) (string, bool) {
	if !isAuthenticationError(apiErr) {
		return "", false
	}
	return formatAuthenticationError(apiErr), true
}

// tryFormatNotFound handles not found errors
func tryFormatNotFound(apiErr *ParsedError) (string, bool) {
	if !isNotFoundError(apiErr) {
		return "", false
	}
	return formatNotFoundError(apiErr), true
}

// tryFormatValidation handles validation errors
func tryFormatValidation(apiErr *ParsedError) (string, bool) {
	if !isValidationError(apiErr) {
		return "", false
	}
	return formatValidationError(apiErr), true
}

// formatDefault handles all other error types
func formatDefault(apiErr *ParsedError) string {
	var msg strings.Builder

	if apiErr.Detail != "" {
		msg.WriteString(apiErr.Detail)
	} else if apiErr.Title != "" {
		msg.WriteString(apiErr.Title)
	} else {
		msg.WriteString("An error occurred")
	}

	appendMetadata(&msg, apiErr)

	return msg.String()
}

// formatAPIError pipes the error through formatters until one handles it
func formatAPIError(apiErr *ParsedError) string {
	formatters := []ErrorFormatter{
		tryFormatPermission,
		tryFormatAuth,
		tryFormatNotFound,
		tryFormatValidation,
	}

	for _, formatter := range formatters {
		if result, handled := formatter(apiErr); handled {
			return result
		}
	}

	return formatDefault(apiErr)
}

// FormatError formats an error for display, extracting structured info if available
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	// Try to parse as structured API error
	if apiErr := parseAPIError(err); apiErr != nil {
		return formatAPIError(apiErr)
	}

	// Fall back to regular error message
	return err.Error()
}
