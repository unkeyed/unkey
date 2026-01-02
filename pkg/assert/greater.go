package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Greater asserts that value 'a' is greater than value 'b'.
// If 'a' is not greater than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate minimum balance
//	if err := assert.Greater(account.Balance, minimumRequired, "Account balance must exceed minimum"); err != nil {
//	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("insufficient balance: %.2f < %.2f"), fault.Public(account.Balance, minimumRequired),
//	        "Insufficient account balance",
//	    ))
//	}
func Greater[T ~int | ~int32 | ~int64 | ~float32 | ~float64 | ~uint | ~uint32 | ~uint64](a, b T, message ...string) error {
	if a > b {
		return nil
	}
	errorMsg := "value is not greater"
	if len(message) > 0 {
		errorMsg = message[0]
	}
	return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
}
