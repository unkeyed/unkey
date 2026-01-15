package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Less asserts that value 'a' is less than value 'b'.
// If 'a' is not less than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate rate limit
//	if err := assert.Less(requestsPerMinute, maxAllowed, "Request rate exceeds limit"); err != nil {
//	    return err
//	}
func Less[T ~int | ~float64](a, b T, message ...string) error {
	if a < b {
		return nil
	}
	errorMsg := "value is not less"
	if len(message) > 0 {
		errorMsg = message[0]
	}
	return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
}
