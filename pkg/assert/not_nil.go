package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// NotNil asserts that the provided value is not nil.
// If the value is nil, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	if err := assert.NotNil(user, "User must be provided"); err != nil {
//	    return fault.Wrap(err, fault.Internal("user object is required"))
//	}
func NotNil(t any, message ...string) error {
	if t == nil {
		errorMsg := "expected not nil"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
