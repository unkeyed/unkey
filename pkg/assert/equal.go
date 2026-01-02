package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Equal asserts that two values of the same comparable type are equal.
// If the values are not equal, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Verify a calculation result
//	if err := assert.Equal(calculateTotal(), 100.0, "Total should be 100.0"); err != nil {
//	    return fault.Wrap(err)
//	}
func Equal[T comparable](a T, b T, message ...string) error {
	if a != b {
		errorMsg := "expected equal"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
