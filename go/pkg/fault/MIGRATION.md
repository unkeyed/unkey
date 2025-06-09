# Migration Guide: Legacy to Primary API

This guide shows how to migrate from the legacy fault API to the new primary API.

## Quick Reference

| Legacy API (Deprecated) | Primary API | Purpose |
|-------------------------|-------------|---------|
| `fault.WithCode(code)` | `fault.Code(code)` | Error classification |
| `fault.WithInternalDesc(msg)` | `fault.Internal(msg)` | Internal debugging info |
| `fault.WithPublicDesc(msg)` | `fault.Public(msg)` | User-facing messages |
| `fault.WithDesc(internal, public)` | `fault.Internal(internal), fault.Public(public)` | Both messages (separate calls) |

## Before and After Examples

### Basic Error Creation

**Before:**
```go
err := fault.New("database error",
    fault.WithCode(codes.URN("DATABASE_ERROR")),
    fault.WithInternalDesc("connection failed to 192.168.1.1:5432"),
    fault.WithPublicDesc("Service temporarily unavailable"),
)
```

**After:**
```go
err := fault.New("database error",
    fault.Code(codes.URN("DATABASE_ERROR")),
    fault.Internal("connection failed to 192.168.1.1:5432"),
    fault.Public("Service temporarily unavailable"),
)
```

### Error Wrapping

**Before:**
```go
return fault.Wrap(err,
    fault.WithCode(codes.Auth.Authentication.KeyNotFound.URN()),
    fault.WithInternalDesc(fmt.Sprintf("failed to find key %s", keyID)),
    fault.WithPublicDesc("The API key was not found"),
)
```

**After:**
```go
return fault.Wrap(err,
    fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
    fault.Internal(fmt.Sprintf("failed to find key %s", keyID)),
    fault.Public("The API key was not found"),
)
```

### Dual Messages (Old Pattern)

**Before:**
```go
return fault.Wrap(err,
    fault.WithDesc(
        "database query failed with timeout",
        "Service temporarily unavailable",
    ),
)
```

**After (Recommended):**
```go
return fault.Wrap(err,
    fault.Internal("database query failed with timeout"),
    fault.Public("Service temporarily unavailable"),
)
```

**After (Alternative - still supported):**
```go
return fault.Wrap(err,
    fault.Desc("database query failed with timeout", "Service temporarily unavailable"),
)
```

### Complex Error Chains

**Before:**
```go
err := fault.Wrap(baseErr,
    fault.WithCode(codes.URN("NETWORK_ERROR")),
    fault.WithInternalDesc("upstream service call failed"),
    fault.WithPublicDesc("Service temporarily unavailable"),
)
err = fault.Wrap(err,
    fault.WithInternalDesc("retry attempt 3/3 failed after 30s"),
    fault.WithPublicDesc("Please try again in a few minutes"),
)
```

**After:**
```go
err := fault.Wrap(baseErr,
    fault.Code(codes.URN("NETWORK_ERROR")),
    fault.Internal("upstream service call failed"),
    fault.Public("Service temporarily unavailable"),
)
err = fault.Wrap(err,
    fault.Internal("retry attempt 3/3 failed after 30s"),
    fault.Public("Please try again in a few minutes"),
)
```

## Character Count Savings

The primary API significantly reduces typing:

- `fault.WithCode` (15 chars) → `fault.Code` (11 chars) = **27% shorter**
- `fault.WithInternalDesc` (22 chars) → `fault.Internal` (15 chars) = **32% shorter**
- `fault.WithPublicDesc` (20 chars) → `fault.Public` (13 chars) = **35% shorter**

## Migration Strategy

### Option 1: Gradual Migration
- Keep existing code unchanged (all legacy APIs are still supported but deprecated)
- Use primary API for all new error handling
- Migrate existing code during regular maintenance

### Option 2: Mass Migration
Use find/replace in your IDE:
1. `fault.WithCode(` → `fault.Code(`
2. `fault.WithInternalDesc(` → `fault.Internal(`
3. `fault.WithPublicDesc(` → `fault.Public(`

### Option 3: Mixed Approach
- Convert `WithDesc` calls to separate `Internal` and `Public` calls for better clarity
- Use find/replace for the other functions
- Focus on new code first, migrate legacy code gradually

## Best Practices with New API

### ✅ Recommended Patterns

```go
// Clear separation of concerns
fault.Wrap(err,
    fault.Code(DATABASE_ERROR),
    fault.Internal("connection pool exhausted"),
    fault.Public("Service temporarily unavailable"),
)

// Internal-only for sensitive operations
fault.Wrap(err,
    fault.Internal(fmt.Sprintf("failed auth for user %s", userID)),
)

// Public-only for validation errors
fault.Wrap(err,
    fault.Public("Please provide a valid email address"),
)
```

### ❌ Avoid These Patterns

```go
// Don't mix primary and legacy APIs unnecessarily
fault.Wrap(err,
    fault.Code(ERROR_CODE),                    // primary
    fault.WithInternalDesc("debug info"),     // legacy (deprecated)
    fault.Public("user message"),             // primary
)

// Don't use WithDesc when you only need one message type
fault.Wrap(err,
    fault.WithDesc("debug info", ""),  // wasteful
)
// Better:
fault.Wrap(err,
    fault.Internal("debug info"),
)
```

## Backward Compatibility

All existing legacy APIs remain fully supported but are deprecated:
- `fault.WithCode` (deprecated: use `fault.Code`)
- `fault.WithInternalDesc` (deprecated: use `fault.Internal`)
- `fault.WithPublicDesc` (deprecated: use `fault.Public`) 
- `fault.WithDesc` (deprecated: use `fault.Internal` + `fault.Public`)

You can mix legacy and primary APIs in the same codebase without issues, but we recommend using only the primary API for new code.