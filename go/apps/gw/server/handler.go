package server

import "context"

// Handler defines the interface for HTTP request handlers in the gateway.
// Implementations receive a Session and return an error if processing fails.
type Handler interface {
	// Handle processes an HTTP request encapsulated by the Session.
	// It should return an error if processing fails.
	Handle(ctx context.Context, sess *Session) error
}

// HandleFunc is a function type that implements the Handler interface.
// It provides a convenient way to create handlers without defining new types.
type HandleFunc func(ctx context.Context, sess *Session) error

// Handle implements the Handler interface for HandleFunc.
func (f HandleFunc) Handle(ctx context.Context, sess *Session) error {
	return f(ctx, sess)
}
