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

### Dual-Message Pattern
Maintain separate internal and public error messages:
```go
fault.Wrap(err,
    fault.WithDesc(
        "database error: connection timeout", // internal message
        "Service temporarily unavailable."     // public message
    ),
)
```


### Error Classification
Tag errors for consistent handling:
```go
var DATABASE_ERROR = fault.Tag("DATABASE_ERROR")

err := fault.New("connection failed",
    fault.WithTag(DATABASE_ERROR),
)

switch fault.GetTag(err) {
	case DATABASE_ERROR:
		// handle
	default:
		// handle
}
```

### Location Tracking
Automatic capture of error locations:
```go
err := fault.New("initial error")         		  // captures location
err = fault.Wrap(err, fault.WithDesc(...))      // captures new location
```

## Philosophy

Fault is built on these principles:

Fault embraces Go’s simplicity and power. By focusing only on essential
abstractions, it keeps your code clean, maintainable, and in harmony with
Go’s design principles.

## Usage

Fault integrates seamlessly with existing Go code:

```go
func ProcessOrder(id string) error {
    order, err := db.FindOrder(id)
    if err != nil {
        return fault.Wrap(err,
            fault.WithTag(DATABASE_ERROR),
            fault.WithDesc(
                fmt.Sprintf("failed to find order %s", id),
                "Order not found",
            ),
        )
    }

    // ... process order ...
}
```
