package fault

import (
	"github.com/unkeyed/unkey/pkg/codes"
)

// Wrapper is a function type that transforms one error into another.
// It's used to build chains of error transformations while preserving
// the original error context.
type Wrapper func(err error) error

// Wrap applies a series of Wrapper functions to an error while capturing
// the call location for debugging purposes. If the input error is nil,
// it returns nil. Multiple wrappers within a single Wrap call are applied
// to a single wrapped instance for efficiency.
//
// Example:
//
//	err := fault.New("database error")
//	withLocationErr := fault.Wrap(baseErr,
//	    fault.Code(DATABASE_ERROR),
//	    fault.Internal("connection failed"),
//	    fault.Public("Service unavailable"),
//	)
func Wrap(err error, wraps ...Wrapper) error {
	if err == nil {
		return nil
	}

	// Create the base wrapped error
	result := &wrapped{
		err:      err,
		location: getLocation(),
		code:     "",
		internal: "",
		public:   "",
	}

	// Apply all wrappers to accumulate information into a single instance
	for _, w := range wraps {
		// nolint:nestif
		if nextErr := w(result); nextErr != nil {
			// If wrapper returns a new error, extract its info and merge
			if nextWrapped, ok := nextErr.(*wrapped); ok {
				if nextWrapped.code != "" && result.code == "" {
					result.code = nextWrapped.code
				}
				if nextWrapped.internal != "" {
					if result.internal == "" {
						result.internal = nextWrapped.internal
					} else {
						result.internal = nextWrapped.internal + ": " + result.internal
					}
				}
				if nextWrapped.public != "" {
					if result.public == "" {
						result.public = nextWrapped.public
					} else {
						result.public = nextWrapped.public + " " + result.public
					}
				}
			}
		}
	}

	return result
}

// Internal creates a new error Wrapper that adds only an internal description to an error.
// Use this for detailed debugging information that should not be exposed to end users.
//
// Example:
//
//	err := fault.Wrap(baseErr, fault.Internal("connection failed to 192.168.1.1:5432"))
func Internal(message string) Wrapper {
	return func(err error) error {
		if err == nil {
			return nil
		}

		return &wrapped{
			err:      err,
			code:     "",
			location: "",
			internal: message,
			public:   "",
		}
	}
}

// Public creates a new error Wrapper that adds only a public description to an error.
// Use this for user-friendly messages that are safe to expose in API responses.
//
// Example:
//
//	err := fault.Wrap(baseErr, fault.Public("Please try again later"))
func Public(message string) Wrapper {
	return func(err error) error {
		if err == nil {
			return nil
		}

		return &wrapped{
			err:      err,
			code:     "",
			location: "",
			internal: "",
			public:   message,
		}
	}
}

// Code creates a new error Wrapper that adds an error code for classification.
// Use this to categorize errors for consistent handling across your application.
//
// Example:
//
//	err := fault.Wrap(baseErr, fault.Code(DATABASE_ERROR))
func Code(code codes.URN) Wrapper {
	return func(err error) error {
		if err == nil {
			return nil
		}

		return &wrapped{
			err:      err,
			code:     code,
			location: "",
			internal: "",
			public:   "",
		}
	}
}
