package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// NotZero asserts that the provided value is not its zero value.
// If the value equals its zero value, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that required fields or dependencies have been
// properly initialized before use.
//
// Example:
//
//	// Validate that a database connection was initialized
//	if err := assert.NotZero(db, "Database connection must be initialized"); err != nil {
//	    return fault.Wrap(err, fault.Internal("database not configured"))
//	}
//
//	// Validate that a struct field was set
//	if err := assert.NotZero(config.Port, "Port must be configured"); err != nil {
//	    return fault.Wrap(err, fault.Internal("missing port configuration"))
//	}
func NotZero[T comparable](value T, message ...string) error {
	var zero T
	if value == zero {
		errorMsg := "value is zero/default"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}

	return nil
}
