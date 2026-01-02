package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// True asserts that a boolean value is true.
// If the value is false, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Verify a precondition
//	if err := assert.True(len(items) > 0, "items cannot be empty"); err != nil {
//	    return fault.Wrap(err)
//	}
func True(value bool, message ...string) error {
	if !value {
		errorMsg := "expected true but got false"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
