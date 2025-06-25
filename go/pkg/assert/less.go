package assert

import (
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Less asserts that value 'a' is less than value 'b'.
// If 'a' is not less than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate rate limit
//	if err := assert.Less(requestsPerMinute, maxAllowed, "Request rate exceeds limit"); err != nil {
//	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("rate limit exceeded: %d > %d"), fault.Public(requestsPerMinute, maxAllowed),
//	        "Too many requests, please try again later",
//	    ))
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
