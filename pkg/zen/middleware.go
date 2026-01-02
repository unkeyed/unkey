package zen

// Middleware transforms one handler into another, typically by adding
// behavior before and/or after the original handler executes.
//
// Middleware is used to implement cross-cutting concerns like logging,
// authentication, error handling, and metrics collection.
type Middleware func(handler HandleFunc) HandleFunc
