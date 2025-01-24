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
