package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// LessOrEqual asserts that value 'a' is less or equal compared to value 'b'.
// If 'a' is not less or equal than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate maximum limit
//	if err := assert.LessOrEqual(requestCount, maxAllowed, "Request count must not exceed maximum"); err != nil {
//	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("limit exceeded: %d > %d"), fault.Public(requestCount, maxAllowed),
//	        "Maximum request limit exceeded",
//	    ))
//	}
func LessOrEqual[T ~int | ~int32 | ~int64 | ~float32 | ~float64](a, b T, message ...string) error {
	if a <= b {
		return nil
	}

	errorMsg := "value is not less or equal"
	if len(message) > 0 {
		errorMsg = message[0]
	}
	return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
}
