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

	// BAD_REQUEST indicates that the client's request was malformed or invalid.
	// This is typically used when request validation fails or when the request
	// cannot be processed due to client-side errors.
	BAD_REQUEST Tag = "BAD_REQUEST"
	// An object was not found in the system.
	NOT_FOUND Tag = "NOT_FOUND"

	UNAUTHORIZED             Tag = "UNAUTHORIZED"
	FORBIDDEN                Tag = "FORBIDDEN"
	INSUFFICIENT_PERMISSIONS Tag = "INSUFFICIENT_PERMISSIONS"

	DATABASE_ERROR Tag = "DATABASE_ERROR"

	INTERNAL_SERVER_ERROR Tag = "INTERNAL_SERVER_ERROR"

	PROTECTED_RESOURCE Tag = "PROTECTED_RESOURCE"

	// An assertion failed during runtime.
	// This tag is used to indicate that a condition that should have been true
	// was false, indicating a programming error.
	//
	// For example we assert that a field on a struct is not empty, but it is.
	ASSERTION_FAILED Tag = "ASSERTION_FAILED"
)

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
		e, ok := err.(*wrapped)
		if ok && e.tag != "" {
			return e.tag
		}
		err = errors.Unwrap(err)
	}

	return UNTAGGED
}

func WithTag(tag Tag) Wrapper {
	return func(err error) error {
		if err == nil {
			return nil
		}

		return &wrapped{
			err:      err,
			tag:      tag,
			location: "",
			internal: "",
			public:   "",
		}
	}
}
