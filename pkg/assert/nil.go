package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Nil asserts that the provided value is nil.
// If the value is not nil, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	err := potentiallyFailingOperation()
//	if assertErr := assert.Nil(err, "Operation should complete without errors"); assertErr != nil {
//	    return fault.Wrap(err, fault.Internal("operation should not fail"))
//	}
func Nil(t any, message ...string) error {
	if t != nil {
		errorMsg := "expected nil"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
