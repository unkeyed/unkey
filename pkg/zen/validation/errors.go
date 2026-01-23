package validation

import (
	"net/http"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TransformErrors converts jsonschema validation errors to the API response format
func TransformErrors(err error, requestID string) openapi.BadRequestErrorResponse {
	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{
				RequestId: requestID,
			},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: err.Error(),
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
				Errors: []openapi.ValidationError{},
			},
		}
	}

	// Use BasicOutput for a flat list of errors
	output := validationErr.BasicOutput()

	errors := collectErrors(output)

	detail := "One or more fields failed validation"
	if len(errors) == 1 {
		detail = errors[0].Message
	}

	return openapi.BadRequestErrorResponse{
		Meta: openapi.Meta{
			RequestId: requestID,
		},
		Error: openapi.BadRequestErrorDetails{
			Title:  "Bad Request",
			Detail: detail,
			Status: http.StatusBadRequest,
			Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
			Errors: errors,
		},
	}
}

// collectErrors extracts validation errors from the output unit hierarchy
func collectErrors(output *jsonschema.OutputUnit) []openapi.ValidationError {
	var errors []openapi.ValidationError

	// Process this unit's error if present
	if output.Error != nil && !output.Valid {
		location := FormatLocation("body", output.InstanceLocation)
		message := output.Error.String()
		fix := suggestFix(output.KeywordLocation, message)

		errors = append(errors, openapi.ValidationError{
			Location: location,
			Message:  message,
			Fix:      fix,
		})
	}

	// Process nested errors
	for i := range output.Errors {
		nested := collectErrors(&output.Errors[i])
		errors = append(errors, nested...)
	}

	return errors
}

// isArrayIndex checks if a string represents an array index
func isArrayIndex(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// suggestFix provides helpful suggestions based on the error type
func suggestFix(keywordLocation, message string) *string {
	// Extract the keyword from the location
	// e.g., "/properties/keyId/type" -> "type"
	parts := strings.Split(keywordLocation, "/")
	keyword := ""
	if len(parts) > 0 {
		keyword = parts[len(parts)-1]
	}

	var fix string

	switch keyword {
	case "required":
		// Extract field name from message if possible
		fix = "Add the missing required field to your request"

	case "type":
		fix = "Ensure the field has the correct data type"

	case "minLength":
		fix = "Provide a longer value"

	case "maxLength":
		fix = "Provide a shorter value"

	case "pattern":
		fix = "Ensure the value matches the required format"

	case "enum":
		fix = "Use one of the allowed values"

	case "minimum":
		fix = "Provide a larger value"

	case "maximum":
		fix = "Provide a smaller value"

	case "minItems":
		fix = "Add more items to the array"

	case "maxItems":
		fix = "Remove some items from the array"

	case "additionalProperties":
		// Try to extract the unknown field name
		fix = "Remove the unknown field from your request"

	default:
		// Don't provide a fix for unknown keywords
		return nil
	}

	return &fix
}

// FormatLocation formats a JSON pointer with an optional prefix for consistent location strings
// e.g., FormatLocation("query", "/0") -> "query[0]"
// e.g., FormatLocation("body", "/roles/0/name") -> "body.roles[0].name"
func FormatLocation(prefix, jsonPointer string) string {
	if jsonPointer == "" || jsonPointer == "/" {
		return prefix
	}

	// Remove leading slash
	path := strings.TrimPrefix(jsonPointer, "/")
	parts := strings.Split(path, "/")

	var result strings.Builder
	result.WriteString(prefix)

	for _, part := range parts {
		if part == "" {
			continue
		}
		if isArrayIndex(part) {
			result.WriteString("[")
			result.WriteString(part)
			result.WriteString("]")
		} else {
			result.WriteString(".")
			result.WriteString(part)
		}
	}

	return result.String()
}
