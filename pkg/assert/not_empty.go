package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// NotEmpty asserts that a string, slice, or map is not empty (has non-zero length).
// If the value is empty, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate required input
//	if err := assert.NotEmpty(request.IDs, "At least one ID must be provided"); err != nil {
//	    return fault.Wrap(err, fault.Internal("IDs cannot be empty"), fault.Public("Please provide at least one ID"))
//	}
func NotEmpty[T ~string | ~[]any | ~map[any]any | []byte](value T, message ...string) error {
	if len(value) == 0 {
		errorMsg := "value is empty"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
