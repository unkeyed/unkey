package zen

// Middleware transforms one handler into another, typically by adding
// behavior before and/or after the original handler executes.
//
// Middleware is used to implement cross-cutting concerns like logging,
// authentication, error handling, and metrics collection.
// The server composes middleware at registration and reuses the returned
// handler concurrently. Keep request state inside the returned function.
type Middleware func(handler HandleFunc) HandleFunc
