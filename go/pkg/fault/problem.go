package fault

// problem represents an error with both user-facing and internal context.
// It implements a dual-message error pattern where public messages can be
// safely exposed to end users while internal details are preserved for
// debugging and logging.
type problem struct {
	// error is the underlying error that caused the problem
	error error

	// publicMessage contains a user-friendly description of the error that is
	// safe to expose in API responses. It should provide actionable guidance for
	// resolving the issue without exposing implementation details.
	publicMessage string

	// internalMessage contains detailed technical information about the error
	// intended for debugging and logging purposes only. This message may contain
	// sensitive implementation details and should never be exposed to end users.
	internalMessage string
}

var _ error = &problem{}

// Error implements the error interface.
// It returns the internal message of the problem, suitable for logging and debugging.
func (p *problem) Error() string {
	return p.internalMessage
}

// Unwrap returns the underlying error that caused this problem.
// This method implements the standard errors.Unwrap interface, allowing the use
// of errors.Is and errors.As functions from the standard library.
func (p *problem) Unwrap() error {
	return p.error
}

// With creates a new error wrapper that adds both internal and public context to an error.
// The wrapper can be used to create a new problem instance that wraps an existing error.
//
// Parameters:
//   - internalMessage: A detailed message for logging and debugging purposes.
//   - publicMessage: A user-friendly message that is safe to expose in API responses.
//
// Returns:
//   - A Wrapper function that can be used to wrap an error with the specified messages.
//
// Example:
//
//	err := someOperation()
//	if err != nil {
//	    return fault.Wrap(err,
//				fault.With("payment failed", "Payment processing failed"),
//			)
//	}
func With(internalMessage string, publicMessage string) Wrapper {
	return func(err error) error {
		return &problem{
			error:           err,
			publicMessage:   publicMessage,
			internalMessage: internalMessage,
		}
	}
}
