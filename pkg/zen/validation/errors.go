package validation

import (
	"fmt"
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
		// Extract keyword and field name once for reuse
		keyword := extractKeyword(output.KeywordLocation)
		message := output.Error.String()

		// Try to get field name from KeywordLocation first, then from message
		fieldName := extractFieldFromKeywordLocation(output.KeywordLocation)
		if fieldName == "" {
			// For required/additionalProperties errors, field name is in the message
			fieldName = extractFieldFromMessage(keyword, message)
		}

		// Build location - append field name if at root for required/additionalProperties
		location := buildLocationWithKeyword("body", output.InstanceLocation, keyword, fieldName)
		fix := suggestFixWithKeyword(keyword, message, fieldName)

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

// extractKeyword extracts the keyword from a jsonschema keyword location
// e.g., "/properties/keyId/type" -> "type"
func extractKeyword(keywordLocation string) string {
	parts := strings.Split(keywordLocation, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractFieldFromKeywordLocation extracts the field name from a jsonschema keyword location
// e.g., "/properties/query/required" -> "query"
// e.g., "/properties/items/0/properties/name/type" -> "name"
func extractFieldFromKeywordLocation(keywordLocation string) string {
	parts := strings.Split(keywordLocation, "/")
	// Look for "properties" and take the next segment (last occurrence)
	lastFieldName := ""
	for i, part := range parts {
		if part == "properties" && i+1 < len(parts) {
			lastFieldName = parts[i+1]
		}
	}
	return lastFieldName
}

// buildLocation creates a user-friendly location string
// For required errors at root level, includes the missing field name
func buildLocation(prefix, instanceLocation, keywordLocation string) string {
	keyword := extractKeyword(keywordLocation)
	fieldName := extractFieldFromKeywordLocation(keywordLocation)
	return buildLocationWithKeyword(prefix, instanceLocation, keyword, fieldName)
}

// buildLocationWithKeyword creates a user-friendly location string using pre-extracted values
func buildLocationWithKeyword(prefix, instanceLocation, keyword, fieldName string) string {
	loc := FormatLocation(prefix, instanceLocation)
	// For required/additionalProperties errors, append the field name to the location
	// This applies at any level, not just root
	if (keyword == "required" || keyword == "additionalProperties") && fieldName != "" {
		return loc + "." + fieldName
	}
	return loc
}

// suggestFix provides helpful suggestions based on the error type
func suggestFix(keywordLocation, message, fieldName string) *string {
	return suggestFixWithKeyword(extractKeyword(keywordLocation), message, fieldName)
}

// suggestFixWithKeyword provides helpful suggestions using a pre-extracted keyword
func suggestFixWithKeyword(keyword, message, fieldName string) *string {

	var fix string

	switch keyword {
	case "required":
		if fieldName != "" {
			fix = fmt.Sprintf("Add the '%s' field to your request body", fieldName)
		} else {
			fix = "Add the missing required field to your request"
		}

	case "type":
		expectedType := extractExpectedType(message)
		if fieldName != "" && expectedType != "" {
			fix = fmt.Sprintf("The '%s' field must be a %s", fieldName, expectedType)
		} else if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field has an incorrect data type", fieldName)
		} else {
			fix = "Ensure the field has the correct data type"
		}

	case "minLength":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field value is too short", fieldName)
		} else {
			fix = "Provide a longer value"
		}

	case "maxLength":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field value is too long", fieldName)
		} else {
			fix = "Provide a shorter value"
		}

	case "pattern":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field does not match the required format", fieldName)
		} else {
			fix = "Ensure the value matches the required format"
		}

	case "enum":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field must be one of the allowed values", fieldName)
		} else {
			fix = "Use one of the allowed values"
		}

	case "minimum":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field value is too small", fieldName)
		} else {
			fix = "Provide a larger value"
		}

	case "maximum":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' field value is too large", fieldName)
		} else {
			fix = "Provide a smaller value"
		}

	case "minItems":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' array has too few items", fieldName)
		} else {
			fix = "Add more items to the array"
		}

	case "maxItems":
		if fieldName != "" {
			fix = fmt.Sprintf("The '%s' array has too many items", fieldName)
		} else {
			fix = "Remove some items from the array"
		}

	case "additionalProperties":
		if fieldName != "" {
			fix = fmt.Sprintf("Remove the '%s' field - it is not allowed", fieldName)
		} else {
			fix = "Remove the unknown field from your request"
		}

	default:
		// Don't provide a fix for unknown keywords
		return nil
	}

	return &fix
}

// extractFieldFromMessage extracts a field name from error messages
// e.g., "missing property 'query'" -> "query"
// e.g., "additionalProperties 'foo' not allowed" -> "foo"
func extractFieldFromMessage(keyword, message string) string {
	switch keyword {
	case "required":
		// Handle "missing property 'fieldName'" format
		if idx := strings.Index(message, "'"); idx != -1 {
			end := strings.Index(message[idx+1:], "'")
			if end != -1 {
				return message[idx+1 : idx+1+end]
			}
		}
	case "additionalProperties":
		// Handle "additionalProperties 'fieldName' not allowed" format
		if idx := strings.Index(message, "'"); idx != -1 {
			end := strings.Index(message[idx+1:], "'")
			if end != -1 {
				return message[idx+1 : idx+1+end]
			}
		}
	}
	return ""
}

// extractExpectedType extracts the expected type from an error message
// e.g., "expected string, got number" -> "string"
// e.g., "got string, want number" -> "number"
func extractExpectedType(message string) string {
	// Handle "expected X, got Y" format
	if strings.HasPrefix(message, "expected ") {
		parts := strings.SplitN(message, ",", 2)
		if len(parts) >= 1 {
			return strings.TrimPrefix(parts[0], "expected ")
		}
	}
	// Handle "got X, want Y" format
	if strings.Contains(message, ", want ") {
		parts := strings.SplitN(message, ", want ", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return ""
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
