package fault

// root represents the initial error in an error chain.
// It contains both the error message and the location where it was created.
type root struct {
	// message contains the error description
	message string
	// location contains the file:line where the error was created
	location string
}

// New creates a new root error with the given message and applies any provided
// wrapper functions. It automatically captures the location where it was called.
//
// Example:
//
//	err := fault.New("failed to connect to database")
func New(message string, wraps ...Wrapper) error {
	var err error
	err = &root{message, getLocation()}

	for _, w := range wraps {
		err = w(err)
	}
	return err
}

// Error implements the error interface by returning the root error message.
func (r *root) Error() string {
	return r.message
}
