package validation

import (
	"net/http"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/config"
	validatorErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/helpers"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/unkeyed/unkey/pkg/fault"
)

// validationError is the library's error type, named locally because this
// package passes it around a lot.
type validationError = validatorErrors.ValidationError

// Validator validates HTTP requests against an OpenAPI specification.
type Validator struct {
	validator validator.Validator

	// properties memoises the property names a schema declares, for the one hint
	// that has to read them out of the rendered schema. See properties.go.
	properties *propertyCache
}

// NewFromBytes creates a Validator from a raw OpenAPI spec.
// Returns an error if the spec cannot be parsed or is itself invalid.
func NewFromBytes(spec []byte) (*Validator, error) {
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create OpenAPI document"))
	}

	v, errors := validator.NewValidator(document, config.WithRegexCache(&sync.Map{}))
	if len(errors) > 0 {
		messages := make([]fault.Wrapper, len(errors))
		for i, e := range errors {
			messages[i] = fault.Internal(e.Error())
		}
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

	return &Validator{validator: v, properties: newPropertyCache()}, nil
}

// Validate checks r against the OpenAPI spec.
// Returns nil when the request is valid; returns a *Result describing
// every failure otherwise.
func (v *Validator) Validate(r *http.Request) *Result {
	valid, errors := v.validator.ValidateHttpRequestSync(r)

	if !valid {
		errors = filterIgnoredSecurityErrors(errors)
		valid = len(errors) == 0
	}

	if valid {
		return nil
	}

	// The library reports one top-level error per part of the request it checked,
	// in a fixed order: path parameters, cookies, headers, query, security, body.
	// All of them are translated, so a request with a bad header and a bad body
	// hears about both, and the summary describes the first.
	collected := make([]ValidationError, 0, len(errors))
	for _, e := range errors {
		collected = append(collected, v.translate(e)...)
	}

	return &Result{
		Detail: detailFor(errors[0]),
		Errors: normalize(collected),
	}
}

// translate turns one top-level validator error into the entries a caller sees.
func (v *Validator) translate(e *validationError) []ValidationError {
	prefix := prefixFor(e)

	if root := schemaFailureRoot(e); root != nil {
		resolver := newPropertyResolver(v.properties, e)
		if errs := fromSchemaFailures(prefix, root, resolver); len(errs) > 0 {
			return errs
		}
	}

	return []ValidationError{describeNonSchemaFailure(prefix, e)}
}

// schemaFailureRoot returns the JSON Schema error tree behind a schema failure.
// The library attaches the same root to every failure it flattened out of that
// tree, and it keeps only the rendered string of each one; the tree still has the
// typed kinds.
func schemaFailureRoot(e *validationError) *jsonschema.ValidationError {
	for _, failure := range e.SchemaValidationErrors {
		if failure != nil && failure.OriginalJsonSchemaError != nil {
			return failure.OriginalJsonSchemaError
		}
	}

	return nil
}

// describeNonSchemaFailure covers the failures that never reach a schema: an
// unparseable body, an unsupported content type, a parameter that is missing or
// could not be decoded into the type the schema declares, a missing credential,
// a path that is not in the specification.
func describeNonSchemaFailure(prefix string, e *validationError) ValidationError {
	switch e.ValidationType {
	case helpers.RequestBodyValidation:
		switch {
		case e.ValidationSubType == helpers.RequestBodyContentType:
			fix := "Send the request body using a content type this operation accepts."
			if types := supportedContentTypes(e.HowToFix); types != "" {
				fix = "Send the request body using one of the content types this operation accepts: " + types + "."
			}

			return newError(prefix, "is not a content type this operation accepts", fix)

		case strings.HasPrefix(e.Reason, reasonBodyUndecodable):
			return newError(locationBody, "is not valid JSON",
				"Check the request body for a syntax error such as a missing quote, comma, or closing brace.")

		case strings.HasPrefix(e.Reason, reasonBodyEmpty):
			return newError(locationBody, "is required", "")

		default:
			return newError(locationBody, fallbackMessage, "")
		}

	case helpers.ParameterValidation:
		if _, predicate, ok := parameterSummary(e); ok {
			return newError(prefix, predicate, "")
		}

		return newError(prefix, parameterFallback, "")

	case helpers.SecurityValidation:
		return newError(prefix, "is missing or does not carry a usable credential", "")

	case helpers.PathValidation, helpers.RequestValidation:
		return newError(locationRequest, "does not match any path and operation in the specification",
			"Check the request path and the HTTP method.")

	default:
		return newError(prefix, fallbackMessage, "")
	}
}

// filterIgnoredSecurityErrors drops OpenAPI security-scheme errors that our
// handlers already produce richer messages for. Specifically:
//
//   - "scheme mismatch" (added in libopenapi-validator v0.13): the handler's
//     bearer parser returns a more useful "missing 'Bearer ' prefix" error.
//
// A missing Authorization header is still surfaced by the validator so that
// the existing 400 invalid_input contract is preserved.
func filterIgnoredSecurityErrors(errs []*validationError) []*validationError {
	filtered := make([]*validationError, 0, len(errs))
	for _, e := range errs {
		if e.ValidationType == helpers.SecurityValidation && e.Reason == "Authorization header had incorrect scheme" {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}
