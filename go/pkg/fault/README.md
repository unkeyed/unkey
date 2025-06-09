<div align="center">
    <h1 align="center">Fault</h1>
</div>


Fault is an error handling package for Go, providing rich context preservation
and safe error reporting in applications. It combines debugging capabilities with
secure user communication patterns.

## Inspiration

I goodartistscopygreatartistssteal'd a lot from [Southclaws/fault](https://github.com/Southclaws/fault).

## Features

Fault addresses common challenges in production error handling:

- Separating internal error details from user-facing messages
- Maintaining complete error context for debugging
- Providing consistent error classification
- Automatic source location tracking
- Safe error chain unwrapping and inspection

### Primary API (Recommended)
Fault provides a clean, concise API for adding error context:

```go
// Internal debugging information (not exposed to users)
fault.Wrap(err,
    fault.Internal("connection failed to 192.168.1.1:5432 after 30s timeout"),
)

// User-facing messages (safe for API responses)
fault.Wrap(err,
    fault.Public("The service is temporarily unavailable. Please try again later."),
)

// Combine as needed
fault.Wrap(err,
    fault.Internal("detailed debug info"),
    fault.Public("user-friendly message"),
    fault.Code(DATABASE_ERROR),
)
```

### Error Classification
Tag errors for consistent handling:
```go
var DATABASE_ERROR = codes.URN("DATABASE_ERROR")

err := fault.New("connection failed",
    fault.Code(DATABASE_ERROR),
)

code, ok := fault.GetCode(err)
if ok && code == DATABASE_ERROR {
    // handle database errors
}
```

### Legacy API (Deprecated)
The following functions are still available for backward compatibility but are deprecated:

```go
// Deprecated: Use fault.Internal() and fault.Public() separately
fault.Wrap(err,
    fault.WithDesc("database error: connection timeout", "Service temporarily unavailable"),
)

// Deprecated: Use fault.Internal() instead
fault.Wrap(err, fault.WithInternalDesc("debug info"))

// Deprecated: Use fault.Public() instead  
fault.Wrap(err, fault.WithPublicDesc("user message"))

// Deprecated: Use fault.Code() instead
fault.Wrap(err, fault.WithCode(DATABASE_ERROR))
```

### Location Tracking
Automatic capture of error locations:
```go
err := fault.New("initial error")         		  // captures location
err = fault.Wrap(err, fault.WithDesc(...))      // captures new location
```

## Philosophy

Fault is built on these principles:

**Concise and Clear**: The primary API (`Internal`, `Public`, `Code`) uses short, 
descriptive names that make error handling fast to write and easy to read.

**Separation of Concerns**: Internal debugging information is kept separate from 
user-facing messages, preventing accidental exposure of sensitive details.

**Go Idioms**: Fault embraces Go's simplicity and power. By focusing only on essential
abstractions, it keeps your code clean, maintainable, and in harmony with
Go's design principles.

## Usage

Fault integrates seamlessly with existing Go code:

```go
func ProcessOrder(id string) error {
    order, err := db.FindOrder(id)
    if err != nil {
        return fault.Wrap(err,
            fault.Code(DATABASE_ERROR),
            fault.Internal(fmt.Sprintf("failed to find order %s", id)),
            fault.Public("Order not found"),
        )
    }

    // ... process order ...
}

func ValidateInput(data string) error {
    if data == "" {
        // Only need user-facing message for validation errors
        return fault.New("validation failed",
            fault.Public("Input cannot be empty"),
        )
    }
    
    if err := complexValidation(data); err != nil {
        // Only need internal debugging for complex operations
        return fault.Wrap(err,
            fault.Internal(fmt.Sprintf("validation failed for input: %q", data)),
        )
    }
    
    return nil
}
```
