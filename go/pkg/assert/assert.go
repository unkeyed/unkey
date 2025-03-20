// unkey/go/pkg/assert/assert.go
package assert

import (
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
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
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Nil asserts that the provided value is nil.
// If the value is not nil, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	err := potentiallyFailingOperation()
//	if assertErr := assert.Nil(err, "Operation should complete without errors"); assertErr != nil {
//	    return fault.Wrap(err, fault.WithDesc("operation should not fail", ""))
//	}
func Nil(t any, message ...string) error {
	if t != nil {
		errorMsg := "expected nil"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// NotNil asserts that the provided value is not nil.
// If the value is nil, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	if err := assert.NotNil(user, "User must be provided"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("user object is required", ""))
//	}
func NotNil(t any, message ...string) error {
	if t == nil {
		errorMsg := "expected not nil"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// True asserts that a boolean value is true.
// If the value is false, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Verify a precondition
//	if err := assert.True(len(items) > 0, "items cannot be empty"); err != nil {
//	    return fault.Wrap(err)
//	}
func True(value bool, message ...string) error {
	if !value {
		errorMsg := "expected true but got false"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// False asserts that a boolean value is false.
// If the value is true, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Safety check
//	if err := assert.False(isShuttingDown, "Cannot perform operation during shutdown"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("cannot perform operation during shutdown", ""))
//	}
func False(value bool, message ...string) error {
	if value {
		errorMsg := "expected false but got true"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Empty asserts that a string, slice, or map is empty (has zero length).
// If the value is not empty, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Verify cleanup was successful
//	if err := assert.Empty(getRemaining(), "No items should remain after cleanup"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("cleanup incomplete", ""))
//	}
func Empty[T ~string | ~[]any | ~map[any]any](value T, message ...string) error {
	if len(value) != 0 {
		errorMsg := "value is not empty"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// NotEmpty asserts that a string, slice, or map is not empty (has non-zero length).
// If the value is empty, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate required input
//	if err := assert.NotEmpty(request.IDs, "At least one ID must be provided"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("IDs cannot be empty", "Please provide at least one ID"))
//	}
func NotEmpty[T ~string | ~[]any | ~map[any]any](value T, message ...string) error {
	if len(value) == 0 {
		errorMsg := "value is empty"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Contains asserts that a string contains a specific substring.
// If the string does not contain the substring, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate email format
//	if err := assert.Contains(email, "@", "Email must contain @ symbol"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("invalid email format", "Please enter a valid email"))
//	}
func Contains(s, substr string, message ...string) error {
	if !strings.Contains(s, substr) {
		errorMsg := "string does not contain substring"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Greater asserts that value 'a' is greater than value 'b'.
// If 'a' is not greater than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate minimum balance
//	if err := assert.Greater(account.Balance, minimumRequired, "Account balance must exceed minimum"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("insufficient balance: %.2f < %.2f", account.Balance, minimumRequired),
//	        "Insufficient account balance",
//	    ))
//	}
func Greater[T ~int | ~int32 | ~int64 | ~float32 | ~float64](a, b T, message ...string) error {
	if a > b {
		return nil

	}
	errorMsg := "value is not greater"
	if len(message) > 0 {
		errorMsg = message[0]
	}
	return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
}

// GreaterOrEqual asserts that value 'a' is greater or equal compared to value 'b'.
// If 'a' is not greater or equal than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate minimum balance
//	if err := assert.GreaterOrEqual(account.Balance, minimumRequired, "Account balance must exceed minimum"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("insufficient balance: %.2f < %.2f", account.Balance, minimumRequired),
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
	return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))

}

// LessOrEqual asserts that value 'a' is less or equal compared to value 'b'.
// If 'a' is not less or equal than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate maximum balance
//	if err := assert.LessOrEqual(account.Balance, minimumRequired, "Account balance must not exceed maximum"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("insufficient balance: %.2f < %.2f", account.Balance, maximumRequired),
//	        "Insufficient account balance",
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
	return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))

}

// Less asserts that value 'a' is less than value 'b'.
// If 'a' is not less than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate rate limit
//	if err := assert.Less(requestsPerMinute, maxAllowed, "Request rate exceeds limit"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("rate limit exceeded: %d > %d", requestsPerMinute, maxAllowed),
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
	return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
}

// InRange asserts that a value is within a specified range (inclusive).
// If the value is outside the range, it returns an error tagged with ASSERTION_FAILED.
//
// Example:
//
//	// Validate age input
//	if err := assert.InRange(age, 18, 120, "Age must be between 18 and 120"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("age %d outside valid range [18-120]", age),
//	        "Please enter a valid age between 18 and 120",
//	    ))
//	}
func InRange[T ~int | ~float64](value, minimum, maximum T, message ...string) error {
	if value < minimum || value > maximum {
		errorMsg := "value is out of range"
		if len(message) > 0 {
			errorMsg = message[0]
		}
		return fault.New(errorMsg, fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}
