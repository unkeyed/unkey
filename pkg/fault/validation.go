package fault

import (
	"errors"
	"slices"
)

// ValidationError describes a public field-level validation failure.
type ValidationError struct {
	Location string
	Message  string
	Fix      *string
}

// Validation attaches public field-level validation details to an error.
func Validation(details ...ValidationError) Wrapper {
	return func(err error) error {
		if err == nil {
			return nil
		}

		return &wrapped{
			err:              err,
			code:             "",
			category:         "",
			location:         "",
			internal:         "",
			public:           "",
			validationErrors: slices.Clone(details),
		}
	}
}

// ValidationErrors returns public field-level validation details attached to err.
func ValidationErrors(err error) []ValidationError {
	details := []ValidationError{}
	for err != nil {
		if wrappedErr, ok := err.(*wrapped); ok {
			details = append(details, wrappedErr.validationErrors...)
		}
		err = errors.Unwrap(err)
	}
	return details
}
