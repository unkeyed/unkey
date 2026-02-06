---
title: fault
description: "provides a clean, concise error handling system designed for"
---

Package fault provides a clean, concise error handling system designed for building robust applications with rich error context and secure user messaging.

The primary API consists of four essential functions:

  - fault.Wrap: Wraps errors while capturing source locations
  - fault.Internal: Adds debugging information (not exposed to users)
  - fault.Public: Adds user-safe messages (safe for API responses)
  - fault.Code: Adds error classification codes

Key features:

  - Clean separation between internal debugging and user-facing messages
  - Concise, ergonomic API with short function names
  - Automatic source location tracking throughout error chains
  - Flexible error classification system
  - Safe error chain unwrapping and inspection

Basic usage:

	err := fault.New("database connection failed")
	if err != nil {
	    return fault.Wrap(err,
	        fault.Code(DATABASE_ERROR),
	        fault.Internal("connection timeout after 30s"),
	        fault.Public("Service temporarily unavailable"),
	    )
	}

Individual message control:

	// Only internal debugging information
	fault.Wrap(err, fault.Internal("detailed debug context"))

	// Only user-facing message
	fault.Wrap(err, fault.Public("Please try again later"))

	// Combine as needed
	fault.Wrap(err,
	    fault.Internal("connection failed to 192.168.1.1:5432"),
	    fault.Public("Database unavailable"),
	    fault.Code(DATABASE_ERROR),
	)

Legacy API: The following functions are deprecated but still available: WithDesc, WithInternalDesc, WithPublicDesc, WithCode

## Functions

### func GetCode

```go
func GetCode(err error) (codes.URN, bool)
```

GetCode examines an error and its chain of wrapped errors to find the first ErrorTag. Returns UNTAGGED if no tag is found or if the error is nil. The search traverses the error chain using errors.Unwrap until either a tag is found or the chain is exhausted.

Example:

		err := errors.New("base error")
		withTag := Tag(DATABASE_ERROR)(err)
		wrapped := fmt.Errorf("wrapped: %w", withTag)
	 code, ok := GetCode(wrapped)
		Output: DATABASE_ERROR, true

### func InternalMessage

```go
func InternalMessage(err error) string
```

InternalMessage extracts all internal messages from an error chain and combines them into a single message for logging purposes. It traverses the error chain from newest to oldest, collecting only the internal descriptions.

This is useful for logging detailed error information without including the full error chain or underlying error messages.

Returns an empty string if:

  - The input error is nil
  - The error is not a wrapped error
  - No internal messages were set in the error chain

### func New

```go
func New(message string, wraps ...Wrapper) error
```

New creates a new error with the given message. The message is stored as an internal error detail, not exposed to end users. Optionally accepts a variadic list of Wrapper functions to modify the error's behavior or add metadata.

Example:

	err := fault.New("database connection failed")
	wrappedErr := fault.New("query error", fault.With("internal message", "user facing message"))

The location where the error was created is automatically captured and stored. The returned error can be further wrapped using fault.Wrap().

### func UserFacingMessage

```go
func UserFacingMessage(err error) string
```

UserFacingMessage extracts all public messages from an error chain and combines them into a single user-safe message. It traverses the error chain from newest to oldest, collecting only the public descriptions that were set using WithDesc.

The function is designed to provide safe, user-friendly error messages that can be returned in API responses or displayed to end users, without exposing sensitive internal details about the error.

The messages are joined with spaces rather than colons (unlike Error()) to create a more readable user-facing message.

Returns an empty string if:

  - The input error is nil
  - The error is not a wrapped error
  - No public messages were set in the error chain

Example usage:

	baseErr := fault.New("internal db error",
		fault.WithDesc(
			"connection timeout to db://internal.example.com",
			"The service is temporarily unavailable",
		))
	wrappedErr := fault.Wrap(baseErr,
		fault.WithDesc(
			"failed to process user request",
			"Please try again later",
		))

	msg := fault.UserFacingMessage(wrappedErr)
	// msg = "Please try again later The service is temporarily unavailable"

Note that only messages set with WithDesc's public parameter are included in the result, maintaining a clear separation between internal error details and user-safe messages.

### func Wrap

```go
func Wrap(err error, wraps ...Wrapper) error
```

Wrap applies a series of Wrapper functions to an error while capturing the call location for debugging purposes. If the input error is nil, it returns nil. Multiple wrappers within a single Wrap call are applied to a single wrapped instance for efficiency.

Example:

	err := fault.New("database error")
	withLocationErr := fault.Wrap(baseErr,
	    fault.Code(DATABASE_ERROR),
	    fault.Internal("connection failed"),
	    fault.Public("Service unavailable"),
	)


## Types

### type Step

```go
type Step struct {
	Message  string
	Location string
}
```

Step represents a single frame in an error chain, capturing the internal message and source location where the error was wrapped. Steps are ordered from the root cause to the outermost wrapper when returned by \[Flatten].

#### func Flatten

```go
func Flatten(err error) []Step
```

Flatten unwraps a chain of wrapped errors and returns each frame as a \[Step]. The returned slice is ordered from root cause to outermost wrapper, making it suitable for logging or displaying error traces. Returns an empty slice if err is nil or not a wrapped error from this package.

### type Wrapper

```go
type Wrapper func(err error) error
```

Wrapper is a function type that transforms one error into another. It's used to build chains of error transformations while preserving the original error context.

#### func Code

```go
func Code(code codes.URN) Wrapper
```

Code creates a new error Wrapper that adds an error code for classification. Use this to categorize errors for consistent handling across your application.

Example:

	err := fault.Wrap(baseErr, fault.Code(DATABASE_ERROR))

#### func Internal

```go
func Internal(message string) Wrapper
```

Internal creates a new error Wrapper that adds only an internal description to an error. Use this for detailed debugging information that should not be exposed to end users.

Example:

	err := fault.Wrap(baseErr, fault.Internal("connection failed to 192.168.1.1:5432"))

#### func Public

```go
func Public(message string) Wrapper
```

Public creates a new error Wrapper that adds only a public description to an error. Use this for user-friendly messages that are safe to expose in API responses.

Example:

	err := fault.Wrap(baseErr, fault.Public("Please try again later"))

