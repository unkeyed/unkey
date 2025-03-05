package assert

import (
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Equal asserts that two values of the same comparable type are equal.
// If the values are not equal, it returns an error tagged with ASSERTION_FAILED.
//
// This function is useful for validating that operations produce expected results
// or that input values match expected values.
//
// Example:
//
//	// Verify a calculation result
//	if err := assert.Equal(calculateTotal(), 100.0); err != nil {
//	    return fault.Wrap(err)
//	}
//
//	// Verify a configuration value
//	if err := assert.Equal(config.Mode, "production"); err != nil {
//	    return fmt.Errorf("incorrect configuration: %w", err)
//	}
func Equal[T comparable](a T, b T) error {
	if a != b {
		return fault.New("expected equal", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Nil asserts that the provided value is nil.
// If the value is not nil, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that certain operations completed successfully
// (when they return nil on success) or that a value does not exist in a particular context.
//
// Example:
//
//	err := potentiallyFailingOperation()
//	if assertErr := assert.Nil(err); assertErr != nil {
//	    return fault.Wrap(err, fault.WithDesc("operation should not fail", ""))
//	}
func Nil(t any) error {
	if t != nil {
		return fault.New("expected nil", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// NotNil asserts that the provided value is not nil.
// If the value is nil, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that required values are present and not nil,
// especially before performing operations that would panic on nil values.
//
// Example:
//
//	if err := assert.NotNil(user); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("user object is required", ""))
//	}
//
//	// Now safe to access user fields
//	processingResult := processUser(user)
func NotNil(t any) error {
	if t == nil {
		return fault.New("expected not nil", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// True asserts that a boolean value is true.
// If the value is false, it returns an error tagged with ASSERTION_FAILED.
// An optional message can be provided to clarify the assertion failure.
//
// This is useful for validating conditions that must be true for the program
// to proceed correctly, especially for preconditions and invariants.
//
// Example:
//
//	// Verify a precondition
//	if err := assert.True(len(items) > 0, "items cannot be empty"); err != nil {
//	    return fault.Wrap(err)
//	}
//
//	// Verify system state
//	if err := assert.True(isInitialized); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("system not initialized", ""))
//	}
func True(value bool, messages ...string) error {
	if !value {
		if len(messages) == 0 {
			messages = []string{"expected true but got false"}
		}
		return fault.New(messages[0], fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// False asserts that a boolean value is false.
// If the value is true, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating conditions that must not be true for
// the program to proceed correctly, especially for safety checks.
//
// Example:
//
//	// Safety check
//	if err := assert.False(isShuttingDown); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("cannot perform operation during shutdown", ""))
//	}
func False(value bool) error {
	if value {
		return fault.New("expected false but got true", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Empty asserts that a string, slice, or map is empty (has zero length).
// If the value is not empty, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that collections are empty when required,
// such as initial states or after clearing operations.
//
// Example:
//
//	// Verify cleanup was successful
//	if err := assert.Empty(getRemaining()); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("cleanup incomplete", ""))
//	}
func Empty[T ~string | ~[]any | ~map[any]any](value T) error {
	if len(value) != 0 {
		return fault.New("value is not empty", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// NotEmpty asserts that a string, slice, or map is not empty (has non-zero length).
// If the value is empty, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that required collections have content,
// particularly for input validation.
//
// Example:
//
//	// Validate required input
//	if err := assert.NotEmpty(request.IDs); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("IDs cannot be empty", "Please provide at least one ID"))
//	}
func NotEmpty[T ~string | ~[]any | ~map[any]any](value T) error {
	if len(value) == 0 {
		return fault.New("value is empty", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Contains asserts that a string contains a specific substring.
// If the string does not contain the substring, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that text contains expected content,
// such as format validation or content checks.
//
// Example:
//
//	// Validate email format
//	if err := assert.Contains(email, "@"); err != nil {
//	    return fault.Wrap(err, fault.WithDesc("invalid email format", "Please enter a valid email"))
//	}
func Contains(s, substr string) error {
	if !strings.Contains(s, substr) {
		return fault.New("string does not contain substring", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Greater asserts that value 'a' is greater than value 'b'.
// If 'a' is not greater than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating numerical constraints, such as minimums or
// sufficient values for operations.
//
// Example:
//
//	// Validate minimum balance
//	if err := assert.Greater(account.Balance, minimumRequired); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("insufficient balance: %.2f < %.2f", account.Balance, minimumRequired),
//	        "Insufficient account balance",
//	    ))
//	}
func Greater[T ~int | ~float64](a, b T) error {
	if a <= b {
		return fault.New("value is not greater", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Less asserts that value 'a' is less than value 'b'.
// If 'a' is not less than 'b', it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating upper bounds or maximum limits.
//
// Example:
//
//	// Validate rate limit
//	if err := assert.Less(requestsPerMinute, maxAllowed); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("rate limit exceeded: %d > %d", requestsPerMinute, maxAllowed),
//	        "Too many requests, please try again later",
//	    ))
//	}
func Less[T ~int | ~float64](a, b T) error {
	if a >= b {
		return fault.New("value is not less", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// InRange asserts that a value is within a specified range (inclusive).
// If the value is outside the range, it returns an error tagged with ASSERTION_FAILED.
//
// This is useful for validating that values fall within acceptable bounds,
// such as configuration parameters or user inputs.
//
// Example:
//
//	// Validate age input
//	if err := assert.InRange(age, 18, 120); err != nil {
//	    return fault.Wrap(err, fault.WithDesc(
//	        fmt.Sprintf("age %d outside valid range [18-120]", age),
//	        "Please enter a valid age between 18 and 120",
//	    ))
//	}
func InRange[T ~int | ~float64](value, minimum, maximum T) error {
	if value < minimum || value > maximum {
		return fault.New("value is out of range", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}
