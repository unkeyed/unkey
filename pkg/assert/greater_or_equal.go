package assert

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// GreaterOrEqual asserts that value 'a' is greater or equal compared to value 'b'.
// If 'a' is not greater or equal than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate minimum balance
//	if err := assert.GreaterOrEqual(account.Balance, minimumRequired, "Account balance must meet minimum"); err != nil {
//	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("insufficient balance: %.2f < %.2f"), fault.Public(account.Balance, minimumRequired),
//	        "Insufficient account balance",
//	    ))
//	}
func GreaterOrEqual[T ~int | ~int32 | ~int64 | ~float32 | ~float64](a, b T, message ...string) error {
	if a >= b {
		return nil
	}

	errorMsg := "value is not greater or equal"
	if len(message) > 0 {
		errorMsg = message[0]
	}
	return fault.New(errorMsg, fault.Code(codes.App.Validation.AssertionFailed.URN()))
}
