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

	// figure out if err is already withLocation or not
	// If it doesn't, we need to capture the location
	_, ok := err.(*withLocation)
	if !ok {
		err = &withLocation{err, getLocation()}
	}
	for _, w := range wraps {
		err = w(err)
	}

	return &withLocation{err, getLocation()}
}
