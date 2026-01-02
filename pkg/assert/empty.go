package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Empty asserts that a string, slice, or map is empty (has zero length).
// If the value is not empty, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Verify cleanup was successful
//	if err := assert.Empty(getRemaining(), "No items should remain after cleanup"); err != nil {
//	    return fault.Wrap(err, fault.Internal("cleanup incomplete"))
//	}
func Empty[T ~string | ~[]any | ~map[any]any](value T, message ...string) error {
	if len(value) != 0 {
		errorMsg := "value is not empty"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
