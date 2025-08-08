package validation

import (
	"context"
	"net/http"
	"strings"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/ctxutil"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"gopkg.in/yaml.v2"
)

type OpenAPIValidator interface {
	// Validate reads the request and validates it against the OpenAPI spec
	//
	// Returns a BadRequestError if the request is invalid that should be
	// marshalled and returned to the client.
	// The second return value is a boolean that is true if the request is valid.
	Validate(r *http.Request) (openapi.BadRequestErrorResponse, bool)
}

type Validator struct {
	validator validator.Validator
}

func New() (*Validator, error) {
	document, err := libopenapi.NewDocument(openapi.Spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create OpenAPI document"))
	}

	v, errors := validator.NewValidator(document)
	if len(errors) > 0 {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fault.Internal(e.Error())
		}
		// nolint:wrapcheck
		return nil, fault.New("failed to create validator", messages...)
	}

	valid, docErrors := v.ValidateDocument()
	if !valid {
		messages := make([]fault.Wrapper, len(docErrors))
		for i, e := range docErrors {
			messages[i] = fault.Internal(e.Message)
		}

		return nil, fault.New("openapi document is invalid", messages...)
	}

	return &Validator{
		validator: v,
	}, nil
}

func (v *Validator) Validate(ctx context.Context, r *http.Request) (openapi.BadRequestErrorResponse, bool) {
	_, validationSpan := tracing.Start(ctx, "openapi.Validate")
	defer validationSpan.End()

	valid, errs := v.validator.ValidateHttpRequest(r)
	if valid {
		// nolint:exhaustruct
		return openapi.BadRequestErrorResponse{}, true
	}

	res := openapi.BadRequestErrorResponse{
		Meta: openapi.Meta{
			RequestId: ctxutil.GetRequestId(r.Context()),
		},
		Error: openapi.BadRequestErrorDetails{
			Title:  "Bad Request",
			Detail: "One or more fields failed validation",
			Status: http.StatusBadRequest,
			Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
			Errors: []openapi.ValidationError{},
		},
	}

	// Process validation errors and filter out nullable field errors
	hasNonNullableErrors := false
	var firstError *errors.ValidationError

	for _, err := range errs {
		// If there are no SchemaValidationErrors (e.g., security errors),
		// this is a non-schema error that should not be filtered
		if len(err.SchemaValidationErrors) == 0 {
			hasNonNullableErrors = true
			if firstError == nil {
				firstError = err
			}
			// Add this error to the response
			res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
				Message:  err.Reason,
				Location: err.ValidationType,
				Fix:      &err.HowToFix,
			})
			continue
		}

		// For each ValidationError with schema errors, check if it has any non-nullable schema errors
		hasNonNullableInThisError := false

		for _, schemaErr := range err.SchemaValidationErrors {
			if !v.isSchemaErrorNullable(schemaErr) {
				hasNonNullableInThisError = true
				hasNonNullableErrors = true

				// Add this non-nullable error to the response
				res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
					Message:  schemaErr.Reason,
					Location: schemaErr.Location,
					Fix:      nil,
				})
			}
		}

		// Keep track of the first error that has non-nullable errors
		if hasNonNullableInThisError && firstError == nil {
			firstError = err
		}
	}

	// If all errors were nullable, validation passes
	if !hasNonNullableErrors {
		return openapi.BadRequestErrorResponse{}, true
	}

	// Set the error detail from the first error with non-nullable issues
	if firstError != nil {
		res.Error.Detail = firstError.Message
	}

	// If no specific errors were added but we have non-nullable errors,
	// add a general error (this shouldn't happen normally)
	if len(res.Error.Errors) == 0 && firstError != nil {
		res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
			Message:  firstError.Reason,
			Location: firstError.ValidationType,
			Fix:      &firstError.HowToFix,
		})
	}

	return res, false
}

// convertToStringMap converts map[interface{}]interface{} to map[string]any
func convertToStringMap(input interface{}) (map[string]any, bool) {
	// Try direct conversion first
	if m, ok := input.(map[string]any); ok {
		return m, true
	}

	// Try interface{} map conversion
	if mapInterface, ok := input.(map[interface{}]interface{}); ok {
		result := make(map[string]any)
		for k, v := range mapInterface {
			if keyStr, ok := k.(string); ok {
				result[keyStr] = v
			}
		}
		return result, true
	}

	return nil, false
}

// isSchemaErrorNullable checks if a single schema validation error is for a nullable field
func (v *Validator) isSchemaErrorNullable(validationErr *errors.SchemaValidationFailure) bool {
	isNullErr := strings.HasPrefix(validationErr.Reason, "got null")
	hasType := strings.HasSuffix(validationErr.Location, "/type")

	if !isNullErr || !hasType {
		return false
	}

	// Parse the location path to navigate nested properties
	// Example: "/properties/credits/properties/remaining/type" -> ["", "properties", "credits", "properties", "remaining", "type"]
	location := strings.TrimSuffix(validationErr.Location, "/type")
	parts := strings.Split(location, "/")

	// We need at least ["", "properties", "fieldName"] for a valid path
	if len(parts) < 3 || parts[1] != "properties" {
		return false
	}

	spec := make(map[string]any)

	// This has to be a valid yaml schema as nothing works without our valid openapi spec anyways.
	err := yaml.Unmarshal([]byte(validationErr.ReferenceSchema), spec)
	if err != nil {
		return false
	}

	// Navigate through the nested structure following the path
	current := spec

	// Start from index 1 to skip the empty string from leading "/"
	// Process pairs of ["properties", "fieldName"]
	for i := 1; i < len(parts)-1; i += 2 {
		if parts[i] != "properties" {
			break
		}

		// Get the properties at current level
		properties, propertiesExist := current["properties"]
		if !propertiesExist {
			return false
		}

		// Convert to map
		propertiesMap, ok := convertToStringMap(properties)
		if !ok {
			return false
		}

		// Get the field at this level
		fieldName := parts[i+1]
		fieldSpec, found := propertiesMap[fieldName]
		if !found {
			return false
		}

		// If this is the last field in the path, check for nullable
		if i+2 >= len(parts) {
			// Convert fieldSpec to map[string]any
			fieldMap, ok := convertToStringMap(fieldSpec)
			if !ok {
				return false
			}

			// Check if nullable exists and is true
			nullable, exists := fieldMap["nullable"]
			if !exists {
				return false
			}

			isNullable, ok := nullable.(bool)
			if !ok || !isNullable {
				return false
			}

			return true
		}

		// Otherwise, move deeper into the structure
		current, ok = convertToStringMap(fieldSpec)
		if !ok {
			return false
		}
	}

	return false
}

// isNullableError checks if ALL schema validation errors in a ValidationError are nullable errors
// Returns true only if every single schema error is a nullable error
func (v *Validator) isNullableError(e *errors.ValidationError) bool {
	if len(e.SchemaValidationErrors) == 0 {
		return false
	}

	// Check if ALL schema validation errors are nullable errors
	for _, validationErr := range e.SchemaValidationErrors {
		if !v.isSchemaErrorNullable(validationErr) {
			// If we find even one non-nullable error, this ValidationError should not be ignored
			return false
		}
	}

	// All errors were nullable errors
	return true
}
