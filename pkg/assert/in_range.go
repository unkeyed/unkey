package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// InRange asserts that a value is within a specified range (inclusive).
// If the value is outside the range, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate age input
//	if err := assert.InRange(age, 18, 120, "Age must be between 18 and 120"); err != nil {
//	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("age %d outside valid range [18-120]"), fault.Public(age),
//	        "Please enter a valid age between 18 and 120",
//	    ))
//	}
func InRange[T ~int | ~float64](value, minimum, maximum T, message ...string) error {
	if value < minimum || value > maximum {
		errorMsg := "value is out of range"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}
	return nil
}
