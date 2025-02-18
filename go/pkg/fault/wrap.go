package fault

// Wrapper is a function type that transforms one error into another.
// It's used to build chains of error transformations while preserving
// the original error context.
type Wrapper func(err error) error

// Wrap applies a series of Wrapper functions to an error while capturing
// the call location for debugging purposes. If the input error is nil,
// it returns nil. If the error isn't already withLocation, it captures the
// current location before applying wrappers.
//
// Example:
//
//	err := fault.New("database error")
//	withLocationErr := fault.Wrap(baseErr,
//	    fault.WithTag(DATABASE_ERROR),
//	    fault.WithDesc("internal", "public"),
//	)
func Wrap(err error, wraps ...Wrapper) error {
	if err == nil {
		return nil
	}

	err = &wrapped{
		err:      err,
		location: getLocation(),
		tag:      "",
		internal: "",
		public:   "",
	}
	for _, w := range wraps {
		err = w(err)
	}

	return err
}

// WithDesc creates a new error Wrapper that adds both internal and public descriptions to an error.
// The internal description is used for logging and debugging, while the public description
// is safe for external exposure (e.g., API responses).
//
// The internal description will be included in the error chain when calling Error(),
// while the public description can be retrieved separately (if implemented).
//
// If the input error is nil, WithDesc returns a nil-returning wrapper to maintain
// error chain integrity.
//
// Example usage:
//
//	// Basic usage
//	err := fault.New("database error",
//		fault.WithDesc(
//			"failed to connect to database at 192.168.1.1:5432",  // internal detail
//			"service temporarily unavailable"                     // public message
//		),
//	)
//
//	// In an error chain
//	baseErr := someDatabase.Connect()
//	wrappedErr := fault.Wrap(baseErr,
//		fault.WithTag(DATABASE_ERROR),
//		fault.WithDesc(
//			fmt.Sprintf("connection timeout after %v", timeout),  // internal detail
//			"database is currently unavailable"                   // public message
//		),
//	)
//
// Parameters:
//   - internal: Detailed error information for logging and debugging
//   - public: User-safe message suitable for external communication
//
// Returns:
//   - Wrapper: A function that wraps an error with the provided descriptions
func WithDesc(internal string, public string) Wrapper {

	return func(err error) error {
		if err == nil {
			return nil
		}

		return &wrapped{
			err:      err,
			tag:      "",
			location: "",
			internal: internal,
			public:   public,
		}
	}

}
