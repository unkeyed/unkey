package types

import "net/http"

// ValidationError represents a single field-level validation failure.
// Fix is non-nil when the validator can suggest a concrete correction.
type ValidationError struct {
	Message  string
	Location string
	Fix      *string
}

// Result holds the validation outcome when a request is invalid.
// A nil *Result means the request passed validation.
type Result struct {
	Detail string
	Errors []ValidationError
}

// Validator validates HTTP requests against an OpenAPI specification.
type Validator interface {
	Validate(r *http.Request) *Result
}

// Factory creates a Validator from a raw OpenAPI specification.
type Factory func(spec []byte) (Validator, error)
