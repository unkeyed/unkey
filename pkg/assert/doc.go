// Package assert provides simple assertion utilities for validating conditions
// and inputs throughout the application. It helps catch programming errors and
// invalid states early by verifying assumptions about data.
//
// When an assertion fails, the package returns a structured error tagged with
// ASSERTION_FAILED to enable consistent error handling. Unlike traditional
// assertion libraries that panic, this package returns errors to allow for
// graceful handling in production environments.
//
// Basic usage:
//
//	if err := assert.NotNil(user); err != nil {
//	    return fault.Wrap(err, fault.Internal("user cannot be nil"), fault.Public("Invalid user"))
//	}
//
//	if err := assert.Equal(count, expected); err != nil {
//	    return fault.Wrap(err, fault.Internal("count mismatch"), fault.Public("Unexpected count"))
//	}
package assert
