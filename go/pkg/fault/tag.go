package fault

import "errors"

// Tag represents a classification label for errors to aid in error handling
// and debugging. It allows categorizing errors into predefined types for consistent
// error classification across the application.
type Tag string

const (
	// UNTAGGED is the default tag for errors that haven't been explicitly withTag.
	// This ensures all errors have at least a basic classification, making error
	// handling more predictable.
	UNTAGGED Tag = "UNTAGGED"
)

// withTag wraps an error with an ErrorTag classification.
// It implements error, Unwrap, and String interfaces for compatibility with
// the standard error handling patterns.
type withTag struct {
	// err holds the original error being wrapped with a tag
	err error
	// tag contains the ErrorTag classification applied to this error
	tag Tag
}

// Error implements the error interface.
// Returns a consistent identifier string rather than the wrapped error message.
// This helps maintain stable error checking across the application.
//
// Example:
//
//	err := &withTag{errors.New("database error"), DATABASE_ERROR}
//	fmt.Println(err.Error()) // Output: "unkey.error.tag"
func (w *withTag) Error() string {
	return "unkey.error.tag"
}

// Cause returns the underlying error that was withTag.
func (w *withTag) Cause() error {
	return w.err
}

// Unwrap returns the underlying error, enabling compatibility with
// errors.Is/As/Unwrap from the standard library.
func (w *withTag) Unwrap() error {
	return w.err
}

// Tag creates an error wrapping function that applies the specified ErrorTag.
// This function is used to consistently tag errors throughout the application.
//
// Example:
//
//	var DATABASE_ERROR = fault.ErrorTag("DATABASE_ERROR")
//	err := fault.New("connection failed", fault.WithTag(DATABASE_ERROR))
//	fmt.Println(fault.GetTag(err)) // Output: DATABASE_ERROR
func WithTag(tag Tag) func(error) error {
	return func(err error) error {
		if err == nil {
			return nil
		}
		return &withTag{err, tag}
	}
}

// GetTag examines an error and its chain of wrapped errors to find the first
// ErrorTag. Returns UNTAGGED if no tag is found or if the error is nil.
// The search traverses the error chain using errors.Unwrap until either a tag
// is found or the chain is exhausted.
//
// Example:
//
//	err := errors.New("base error")
//	withTag := Tag(DATABASE_ERROR)(err)
//	wrapped := fmt.Errorf("wrapped: %w", withTag)
//	fmt.Println(GetTag(wrapped)) // Output: DATABASE_ERROR
func GetTag(err error) Tag {
	if err == nil {
		return UNTAGGED
	}

	for err != nil {
		e, ok := err.(*withTag)
		if ok {
			return e.tag
		}
		err = errors.Unwrap(err)
	}

	return UNTAGGED
}
