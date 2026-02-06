---
title: assert
description: "provides simple assertion utilities for validating conditions"
---

Package assert provides simple assertion utilities for validating conditions and inputs throughout the application. It helps catch programming errors and invalid states early by verifying assumptions about data.

When an assertion fails, the package returns a structured error tagged with ASSERTION\_FAILED to enable consistent error handling. Unlike traditional assertion libraries that panic, this package returns errors to allow for graceful handling in production environments.

Basic usage:

	if err := assert.NotNil(user); err != nil {
	    return fault.Wrap(err, fault.Internal("user cannot be nil"), fault.Public("Invalid user"))
	}

	if err := assert.Equal(count, expected); err != nil {
	    return fault.Wrap(err, fault.Internal("count mismatch"), fault.Public("Unexpected count"))
	}

## Functions

### func All

```go
func All(errs ...error) error
```

All combines multiple assertion checks into a single operation, returning the first encountered error (if any). This function allows for concise validation of multiple conditions without repetitive error checking.

All stops checking at the first error it encounters.

Example:

	err := assert.All(
	    assert.NotNil(user),
	    assert.NotEmpty(user.ID),
	    assert.True(user.IsActive, "user must be active"),
	)
	if err != nil {
	    return fault.Wrap(err, fault.Internal("invalid user"), fault.Public("User validation failed"))
	}

### func Contains

```go
func Contains(s, substr string, message ...string) error
```

Contains asserts that a string contains a specific substring. If the string does not contain the substring, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate email format
	if err := assert.Contains(email, "@", "Email must contain @ symbol"); err != nil {
	    return fault.Wrap(err, fault.Internal("invalid email format"), fault.Public("Please enter a valid email"))
	}

### func Empty

```go
func Empty[T ~string | ~[]any | ~map[any]any](value T, message ...string) error
```

Empty asserts that a string, slice, or map is empty (has zero length). If the value is not empty, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Verify cleanup was successful
	if err := assert.Empty(getRemaining(), "No items should remain after cleanup"); err != nil {
	    return fault.Wrap(err, fault.Internal("cleanup incomplete"))
	}

### func Equal

```go
func Equal[T comparable](a T, b T, message ...string) error
```

Equal asserts that two values of the same comparable type are equal. If the values are not equal, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Verify a calculation result
	if err := assert.Equal(calculateTotal(), 100.0, "Total should be 100.0"); err != nil {
	    return fault.Wrap(err)
	}

### func False

```go
func False(value bool, message ...string) error
```

False asserts that a boolean value is false. If the value is true, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Safety check
	if err := assert.False(isShuttingDown, "Cannot perform operation during shutdown"); err != nil {
	    return fault.Wrap(err, fault.Internal("cannot perform operation during shutdown"))
	}

### func Greater

```go
func Greater[T ~int | ~int32 | ~int64 | ~float32 | ~float64 | ~uint | ~uint32 | ~uint64](a, b T, message ...string) error
```

Greater asserts that value 'a' is greater than value 'b'. If 'a' is not greater than 'b', it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate minimum balance
	if err := assert.Greater(account.Balance, minimumRequired, "Account balance must exceed minimum"); err != nil {
	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("insufficient balance: %.2f < %.2f"), fault.Public(account.Balance, minimumRequired),
	        "Insufficient account balance",
	    ))
	}

### func GreaterOrEqual

```go
func GreaterOrEqual[T ~int | ~int32 | ~int64 | ~float32 | ~float64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](a, b T, message ...string) error
```

GreaterOrEqual asserts that value 'a' is greater or equal compared to value 'b'. If 'a' is not greater or equal than 'b', it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate minimum balance
	if err := assert.GreaterOrEqual(account.Balance, minimumRequired, "Account balance must meet minimum"); err != nil {
	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("insufficient balance: %.2f < %.2f"), fault.Public(account.Balance, minimumRequired),
	        "Insufficient account balance",
	    ))
	}

### func InRange

```go
func InRange[T ~int | ~float64](value, minimum, maximum T, message ...string) error
```

InRange asserts that a value is within a specified range (inclusive). If the value is outside the range, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate age input
	if err := assert.InRange(age, 18, 120, "Age must be between 18 and 120"); err != nil {
	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("age %d outside valid range [18-120]"), fault.Public(age),
	        "Please enter a valid age between 18 and 120",
	    ))
	}

### func Less

```go
func Less[T ~int | ~float64](a, b T, message ...string) error
```

Less asserts that value 'a' is less than value 'b'. If 'a' is not less than 'b', it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate rate limit
	if err := assert.Less(requestsPerMinute, maxAllowed, "Request rate exceeds limit"); err != nil {
	    return err
	}

### func LessOrEqual

```go
func LessOrEqual[T ~int | ~int32 | ~int64 | ~float32 | ~float64](a, b T, message ...string) error
```

LessOrEqual asserts that value 'a' is less or equal compared to value 'b'. If 'a' is not less or equal than 'b', it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate maximum limit
	if err := assert.LessOrEqual(requestCount, maxAllowed, "Request count must not exceed maximum"); err != nil {
	    return fault.Wrap(err, fault.Internal(//	        fmt.Sprintf("limit exceeded: %d > %d"), fault.Public(requestCount, maxAllowed),
	        "Maximum request limit exceeded",
	    ))
	}

### func Nil

```go
func Nil(t any, message ...string) error
```

Nil asserts that the provided value is nil. If the value is not nil, it returns an error tagged with ASSERTION\_FAILED.

Example:

	err := potentiallyFailingOperation()
	if assertErr := assert.Nil(err, "Operation should complete without errors"); assertErr != nil {
	    return fault.Wrap(err, fault.Internal("operation should not fail"))
	}

### func NotEmpty

```go
func NotEmpty[T ~string | ~[]any | ~[]string | ~map[any]any | []byte](value T, message ...string) error
```

NotEmpty asserts that a string, slice, or map is not empty (has non-zero length). If the value is empty, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Validate required input
	if err := assert.NotEmpty(request.IDs, "At least one ID must be provided"); err != nil {
	    return fault.Wrap(err, fault.Internal("IDs cannot be empty"), fault.Public("Please provide at least one ID"))
	}

### func NotEqual

```go
func NotEqual[T comparable](a T, b T, message ...string) error
```

NotEqual asserts that two values of the same comparable type are not equal. If the values are equal, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Verify values are different
	if err := assert.NotEqual(userID, adminID, "User should not be admin"); err != nil {
	    return fault.Wrap(err)
	}

### func NotNil

```go
func NotNil(t any, message ...string) error
```

NotNil asserts that the provided value is not nil. If the value is nil, it returns an error tagged with ASSERTION\_FAILED.

Example:

	if err := assert.NotNil(user, "User must be provided"); err != nil {
	    return fault.Wrap(err, fault.Internal("user object is required"))
	}

### func NotNilAndNotZero

```go
func NotNilAndNotZero[T comparable](value T, message ...string) error
```

NotNilAndNotZero asserts that the provided value is both not nil and not its zero value. This is useful for validating pointer types and interface values that should be properly initialized.

For most types, this is equivalent to just NotZero since nil pointers, nil interfaces, and nil slices/maps are all zero values. However, this function provides clearer semantics when you specifically want to check both conditions.

Example:

	// Validate that a database interface is both provided and initialized
	if err := assert.NotNilAndNotZero(db, "Database must be provided and initialized"); err != nil {
	    return fault.Wrap(err, fault.Internal("database validation failed"))
	}

	// Validate that a config pointer is both not nil and has values set
	if err := assert.NotNilAndNotZero(config, "Config must be provided and initialized"); err != nil {
	    return fault.Wrap(err, fault.Internal("config validation failed"))
	}

### func NotZero

```go
func NotZero[T comparable](value T, message ...string) error
```

NotZero asserts that the provided value is not its zero value. If the value equals its zero value, it returns an error tagged with ASSERTION\_FAILED.

This is useful for validating that required fields or dependencies have been properly initialized before use.

Example:

	// Validate that a database connection was initialized
	if err := assert.NotZero(db, "Database connection must be initialized"); err != nil {
	    return fault.Wrap(err, fault.Internal("database not configured"))
	}

	// Validate that a struct field was set
	if err := assert.NotZero(config.Port, "Port must be configured"); err != nil {
	    return fault.Wrap(err, fault.Internal("missing port configuration"))
	}

### func Some

```go
func Some(errs ...error) error
```

Some checks multiple assertions and returns nil if at least one assertion passes. If all assertions fail, it returns the first error encountered.

Unlike All, which ensures all assertions pass, Some only requires that at least one assertion succeeds. This is useful for validating conditions where multiple alternatives are acceptable.

Example:

	// Check if user has any required role
	err := assert.Some(
	    assert.Equal(user.Role, "admin"),
	    assert.Equal(user.Role, "editor"),
	    assert.Equal(user.Role, "manager"),
	)
	if err != nil {
	    return fault.Wrap(err, fault.Internal("insufficient permissions"), fault.Public("User lacks required role"))
	}

### func True

```go
func True(value bool, message ...string) error
```

True asserts that a boolean value is true. If the value is false, it returns an error tagged with ASSERTION\_FAILED.

Example:

	// Verify a precondition
	if err := assert.True(len(items) > 0, "items cannot be empty"); err != nil {
	    return fault.Wrap(err)
	}

