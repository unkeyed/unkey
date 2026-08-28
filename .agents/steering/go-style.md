# Go Style Conventions

## Struct Placement

All struct definitions — including internal helper types used only within a single function — must be declared at the package level (top of the file), not inline within function bodies.

This keeps type declarations in a predictable location and makes it easy to scan a file's data model at a glance.

```go
// Good: struct defined at package level
type exchangeResult struct {
    browserSessionID string
    expiresAt        int64
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
    result, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (exchangeResult, error) {
        // ...
    })
    // ...
}
```

```go
// Bad: struct defined inline inside a function
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
    type exchangeResult struct {
        browserSessionID string
        expiresAt        int64
    }

    result, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (exchangeResult, error) {
        // ...
    })
    // ...
}
```

## Type Aliases for Request/Response

Handler packages define `Request` and `Response` type aliases at the top of the file, referencing the generated OpenAPI types. This gives tests and other consumers a stable, short name to use.

```go
type (
    Request  = openapi.V2PortalCreateSessionRequestBody
    Response = openapi.V2PortalCreateSessionResponseBody
)
```
