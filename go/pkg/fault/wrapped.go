package fault

import (
	"fmt"
	"runtime"
	"strings"
)

// wrapped represents an error with additional contextual information.
// It implements both the error interface and provides extended functionality for
// tracking error chains, locations, and separate internal/public messaging.
//
// The wrapped error maintains a link to its underlying error (if any) as well as
// the source location where it was created. It differentiates between public-facing
// messages that are safe to expose to end users and internal details meant for
// debugging and logging.
//
// This type should not be created directly - use fault.New() or fault.Wrap() instead.
type wrapped struct {
	// err is the underlying error being wrapped
	err error
	// location contains the file and line number where the error was link
	location string

	tag Tag

	// public contains a user-friendly description of the error that is
	// safe to expose in API responses. It should provide actionable guidance for
	// resolving the issue without exposing implementation details.
	public string

	// internal contains detailed technical information about the error
	// intended for debugging and logging purposes only. This message may contain
	// sensitive implementation details and should never be exposed to end users.
	internal string
}

// New creates a new error with the given message. The message is stored as an
// internal error detail, not exposed to end users. Optionally accepts a
// variadic list of Wrapper functions to modify the error's behavior or add
// metadata.
//
// Example:
//
//	err := fault.New("database connection failed")
//	wrappedErr := fault.New("query error", fault.With("internal message", "user facing message"))
//
// The location where the error was created is automatically captured and stored.
// The returned error can be further wrapped using fault.Wrap().
func New(message string, wraps ...Wrapper) error {
	var err error
	err = &wrapped{
		err:      nil,
		tag:      "",
		location: getLocation(),
		public:   "",
		internal: message,
	}

	for _, w := range wraps {
		err = Wrap(err, w)
	}
	return err
}

// Error implements the error interface by returning the underlying error messages
// in the order they were added to the chain, from oldest to newest.
//
// Example:
//
//	baseErr := fault.New("initial error")
//	link := fault.Wrap(baseErr, fault.With })
//	fmt.Println(link.Error()) // Output will contain all messages in chain
func (w *wrapped) Error() string {
	errs := []string{}
	current := w

	for current != nil {
		if current.internal != "" {
			errs = append(errs, current.internal)
		}

		// Check if there's a next error in the chain
		if current.err == nil {
			break
		}

		// Try to get the next wrapped error
		next, ok := current.err.(*wrapped)
		if !ok {
			// If it's not a wrapped error, add the error string and break
			errs = append(errs, current.err.Error())
			break
		}
		current = next
	}

	return strings.Join(errs, ": ")
}

func (w *wrapped) Unwrap() error {
	return w.err
}

// getLocation returns a string containing the file name and line number
// where it was called. It skips 3 frames in the call stack to get the
// actual location where the error was created rather than the internal
// function calls.
//
// The returned string is in the format "filename:linenumber"
//
// Example:
//
//	loc := getLocation() // might return "main.go:42"
func getLocation() string {
	pc := make([]uintptr, 1)
	runtime.Callers(3, pc)
	cf := runtime.CallersFrames(pc)
	f, _ := cf.Next()

	return fmt.Sprintf("%s:%d", f.File, f.Line)
}

// UserFacingMessage extracts all public messages from an error chain and combines them
// into a single user-safe message. It traverses the error chain from newest to oldest,
// collecting only the public descriptions that were set using WithDesc.
//
// The function is designed to provide safe, user-friendly error messages that can be
// returned in API responses or displayed to end users, without exposing sensitive
// internal details about the error.
//
// The messages are joined with spaces rather than colons (unlike Error()) to create
// a more readable user-facing message.
//
// Returns an empty string if:
//   - The input error is nil
//   - The error is not a wrapped error
//   - No public messages were set in the error chain
//
// Example usage:
//
//	baseErr := fault.New("internal db error",
//		fault.WithDesc(
//			"connection timeout to db://internal.example.com",
//			"The service is temporarily unavailable",
//		))
//	wrappedErr := fault.Wrap(baseErr,
//		fault.WithDesc(
//			"failed to process user request",
//			"Please try again later",
//		))
//
//	msg := fault.UserFacingMessage(wrappedErr)
//	// msg = "Please try again later The service is temporarily unavailable"
//
// Note that only messages set with WithDesc's public parameter are included in
// the result, maintaining a clear separation between internal error details and
// user-safe messages.
func UserFacingMessage(err error) string {
	if err == nil {
		return ""
	}

	current, ok := err.(*wrapped)
	if !ok {
		return ""
	}

	errs := []string{}

	for current != nil {
		if current.public != "" {
			errs = append(errs, current.public)
		}

		// Check if there's a next error in the chain
		if current.err == nil {
			break
		}

		// Try to get the next wrapped error
		next, ok := current.err.(*wrapped)
		if !ok {
			// if it's not a wrapepd error, then we don't have any more public messages
			// and can stop looking.
			break
		}
		current = next
	}

	return strings.Join(errs, " ")

}
