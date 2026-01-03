package fault

import (
	"errors"

	"github.com/unkeyed/unkey/pkg/codes"
)

// GetTag examines an error and its chain of wrapped errors to find the first
// ErrorTag. Returns UNTAGGED if no tag is found or if the error is nil.
// The search traverses the error chain using errors.Unwrap until either a tag
// is found or the chain is exhausted.
//
// Example:
//
//		err := errors.New("base error")
//		withTag := Tag(DATABASE_ERROR)(err)
//		wrapped := fmt.Errorf("wrapped: %w", withTag)
//	 code, ok := GetCode(wrapped)
//		Output: DATABASE_ERROR, true
func GetCode(err error) (codes.URN, bool) {
	if err == nil {
		return "", false
	}

	for err != nil {
		e, ok := err.(*wrapped)
		if ok && e.code != "" {
			return e.code, true
		}
		err = errors.Unwrap(err)
	}

	return "", false
}
