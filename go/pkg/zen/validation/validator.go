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
			Type:   "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input",
			Errors: []openapi.ValidationError{},
		},
	}

	// Collect non-nullable errors
	var nonNullableErrors []*errors.ValidationError
	for _, err := range errs {
		if !v.isNullableError(err) {
			nonNullableErrors = append(nonNullableErrors, err)
		}
	}

	// If all errors are nullable, consider the request valid
	if len(nonNullableErrors) == 0 {
		return openapi.BadRequestErrorResponse{}, true
	}

	// Process the first non-nullable error
	err := nonNullableErrors[0]
	res.Error.Detail = err.Message
	for _, verr := range err.SchemaValidationErrors {
		res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
			Message:  verr.Reason,
			Location: verr.Location,
			Fix:      nil,
		})
	}

	if len(res.Error.Errors) == 0 {
		res.Error.Errors = append(res.Error.Errors, openapi.ValidationError{
			Message:  err.Reason,
			Location: err.ValidationType,
			Fix:      &err.HowToFix,
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

func (v *Validator) isNullableError(e *errors.ValidationError) bool {
	if len(e.SchemaValidationErrors) == 0 {
		return false
	}

	for _, validationErr := range e.SchemaValidationErrors {
		isNullErr := strings.HasPrefix(validationErr.Reason, "got null")
		hasType := strings.HasSuffix(validationErr.Location, "/type")

		if !isNullErr || !hasType {
			continue
		}

		// "/properties/name/type" -> extract "name"
		location := strings.TrimSuffix(validationErr.Location, "/type")
		parts := strings.Split(location, "/")
		if len(parts) < 3 || parts[1] != "properties" {
			continue
		}
		fieldName := parts[2]

		spec := make(map[string]any)

		// This has to be a valid yaml schema as nothing works without our valid openapi spec anyways.
		err := yaml.Unmarshal([]byte(validationErr.ReferenceSchema), spec)
		if err != nil {
			return false
		}

		properties, propertiesExist := spec["properties"]
		if !propertiesExist {
			return false
		}

		// Convert properties to map[string]any
		propertiesMap, ok := convertToStringMap(properties)
		if !ok {
			return false
		}

		// Get the field spec directly
		fieldSpec, found := propertiesMap[fieldName]
		if !found {
			return false
		}

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

		// Its nullable just ignore the error its fine.
		return true
	}

	return false
}
