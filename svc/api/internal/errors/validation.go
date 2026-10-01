package errors

import (
	"errors"

	"github.com/unkeyed/unkey/svc/api/openapi"
)

// ValidationError describes a public field-level request validation failure.
type ValidationError = openapi.ValidationError

type validationError struct {
	err    error
	detail ValidationError
}

// WithValidationError attaches a public validation detail to err.
func WithValidationError(err error, detail ValidationError) error {
	if err == nil {
		return nil
	}
	return &validationError{err: err, detail: detail}
}

// ValidationErrors returns public validation details attached to err.
func ValidationErrors(err error) []ValidationError {
	var detailed interface {
		ValidationErrors() []ValidationError
	}
	if !errors.As(err, &detailed) {
		return []ValidationError{}
	}
	return detailed.ValidationErrors()
}

func (e *validationError) Error() string {
	return e.err.Error()
}

func (e *validationError) Unwrap() error {
	return e.err
}

func (e *validationError) ValidationErrors() []ValidationError {
	return []ValidationError{e.detail}
}
