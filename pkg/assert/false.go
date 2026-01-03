package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// False asserts that a boolean value is false.
// If the value is true, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Safety check
//	if err := assert.False(isShuttingDown, "Cannot perform operation during shutdown"); err != nil {
//	    return fault.Wrap(err, fault.Internal("cannot perform operation during shutdown"))
//	}
func False(value bool, message ...string) error {
	if value {
		errorMsg := "expected false but got true"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
