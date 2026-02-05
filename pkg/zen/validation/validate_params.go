package validation

import (
	"net/http"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// extractTypeInfo extracts itemType and propertyTypes from a TypedSchema for use with ParseByStyle
func extractTypeInfo(ts *TypedSchema) (itemType SchemaType, propertyTypes map[string]SchemaType) {
	itemType = SchemaTypeString // default for array items
	propertyTypes = nil

	if ts == nil {
		return
	}

	if ts.Items != nil {
		itemType = ts.Items.Type
	}

	if ts.Properties != nil {
		propertyTypes = make(map[string]SchemaType)
		for name, prop := range ts.Properties {
			if prop != nil {
				propertyTypes[name] = prop.Type
			}
		}
	}

	return
}

// validatePathParams validates path parameters against their schemas
func (v *Validator) validatePathParams(pathParams map[string]string, params []CompiledParameter) []openapi.ValidationError {
	var errors []openapi.ValidationError

	for _, param := range params {
		value, exists := pathParams[param.Name]

		if !exists || value == "" {
			errors = append(errors, openapi.ValidationError{
				Location: "path." + param.Name,
				Message:  "required path parameter is missing",
				Fix:      ptr.P("Provide the required path parameter: " + param.Name),
			})
			continue
		}

		if param.Schema != nil {
			itemType, propertyTypes := extractTypeInfo(param.TypedSchema)
			parsedValue := ParseByStyle(param.Style, param.Explode, []string{value}, param.SchemaType, nil, param.Name, itemType, propertyTypes)
			if err := param.Schema.Validate(parsedValue.ToAny()); err != nil {
				errors = append(errors, v.transformParamError(err, "path", param.Name)...)
			}
		}
	}

	return errors
}

// validateQueryParams validates query parameters against their schemas
func (v *Validator) validateQueryParams(r *http.Request, params []CompiledParameter) []openapi.ValidationError {
	var errors []openapi.ValidationError
	query := r.URL.Query()

	for _, param := range params {
		// DeepObject params use format like filter[name]=foo&filter[age]=30
		if param.Style == "deepObject" {
			itemType, propertyTypes := extractTypeInfo(param.TypedSchema)
			parsedValue := ParseByStyle(param.Style, param.Explode, nil, param.SchemaType, query, param.Name, itemType, propertyTypes)

			if param.Required && parsedValue.IsNil() {
				errors = append(errors, openapi.ValidationError{
					Location: "query." + param.Name,
					Message:  "required parameter is missing",
					Fix:      ptr.P("Add the required query parameter: " + param.Name),
				})
				continue
			}

			if parsedValue.IsNil() {
				continue
			}

			if param.Schema != nil {
				if err := param.Schema.Validate(parsedValue.ToAny()); err != nil {
					errors = append(errors, v.transformParamError(err, "query", param.Name)...)
				}
			}
			continue
		}

		values, exists := query[param.Name]

		if param.Required && (!exists || len(values) == 0 || (values[0] == "" && !param.AllowEmptyValue)) {
			errors = append(errors, openapi.ValidationError{
				Location: "query." + param.Name,
				Message:  "required parameter is missing",
				Fix:      ptr.P("Add the required query parameter: " + param.Name),
			})
			continue
		}

		if !exists || len(values) == 0 {
			continue
		}

		if !param.AllowEmptyValue && len(values) == 1 && values[0] == "" {
			errors = append(errors, openapi.ValidationError{
				Location: "query." + param.Name,
				Message:  "parameter value cannot be empty",
				Fix:      ptr.P("Provide a non-empty value for parameter: " + param.Name),
			})
			continue
		}

		if param.Schema != nil {
			itemType, propertyTypes := extractTypeInfo(param.TypedSchema)
			parsedValue := ParseByStyle(param.Style, param.Explode, values, param.SchemaType, query, param.Name, itemType, propertyTypes)
			if err := param.Schema.Validate(parsedValue.ToAny()); err != nil {
				errors = append(errors, v.transformParamError(err, "query", param.Name)...)
			}
		}
	}

	return errors
}

// validateHeaderParams validates header parameters against their schemas
func (v *Validator) validateHeaderParams(r *http.Request, params []CompiledParameter) []openapi.ValidationError {
	var errors []openapi.ValidationError

	for _, param := range params {
		value := r.Header.Get(param.Name)

		if param.Required && value == "" {
			errors = append(errors, openapi.ValidationError{
				Location: "header." + param.Name,
				Message:  "required header is missing",
				Fix:      ptr.P("Add the required header: " + param.Name),
			})
			continue
		}

		if value == "" {
			continue
		}

		if param.Schema != nil {
			itemType, propertyTypes := extractTypeInfo(param.TypedSchema)
			parsedValue := ParseByStyle(param.Style, param.Explode, []string{value}, param.SchemaType, nil, param.Name, itemType, propertyTypes)
			if err := param.Schema.Validate(parsedValue.ToAny()); err != nil {
				errors = append(errors, v.transformParamError(err, "header", param.Name)...)
			}
		}
	}

	return errors
}

// validateCookieParams validates cookie parameters against their schemas
func (v *Validator) validateCookieParams(r *http.Request, params []CompiledParameter) []openapi.ValidationError {
	var errors []openapi.ValidationError

	for _, param := range params {
		cookie, err := r.Cookie(param.Name)
		value := ""
		if err == nil {
			value = cookie.Value
		}

		if param.Required && value == "" {
			errors = append(errors, openapi.ValidationError{
				Location: "cookie." + param.Name,
				Message:  "required cookie is missing",
				Fix:      ptr.P("Add the required cookie: " + param.Name),
			})
			continue
		}

		if value == "" {
			continue
		}

		if param.Schema != nil {
			itemType, propertyTypes := extractTypeInfo(param.TypedSchema)
			parsedValue := ParseByStyle(param.Style, param.Explode, []string{value}, param.SchemaType, nil, param.Name, itemType, propertyTypes)
			if err := param.Schema.Validate(parsedValue.ToAny()); err != nil {
				errors = append(errors, v.transformParamError(err, "cookie", param.Name)...)
			}
		}
	}

	return errors
}

// transformParamError converts a schema validation error to ValidationError format
func (v *Validator) transformParamError(err error, location, paramName string) []openapi.ValidationError {
	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []openapi.ValidationError{
			{
				Location: location + "." + paramName,
				Message:  err.Error(),
				Fix:      nil,
			},
		}
	}

	output := validationErr.BasicOutput()
	return collectParamErrors(output, location, paramName)
}

// collectParamErrors extracts validation errors for a parameter
func collectParamErrors(output *jsonschema.OutputUnit, location, paramName string) []openapi.ValidationError {
	var errors []openapi.ValidationError

	if output.Error != nil && !output.Valid {
		loc := FormatLocation(location+"."+paramName, output.InstanceLocation)

		message := output.Error.String()
		fieldName := extractFieldFromKeywordLocation(output.KeywordLocation)
		if fieldName == "" {
			fieldName = paramName
		}
		fix := suggestFixWithKeyword(extractKeyword(output.KeywordLocation), message, fieldName)

		errors = append(errors, openapi.ValidationError{
			Location: loc,
			Message:  message,
			Fix:      fix,
		})
	}

	for i := range output.Errors {
		nested := collectParamErrors(&output.Errors[i], location, paramName)
		errors = append(errors, nested...)
	}

	return errors
}
