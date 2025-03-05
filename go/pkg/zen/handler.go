package zen

// Handler defines the interface for HTTP request handlers in the Zen framework.
// Implementations receive a Session and return an error if processing fails.
type Handler interface {
	// Handle processes an HTTP request encapsulated by the Session.
	// It should return an error if processing fails.
	Handle(sess *Session) error
}

// HandleFunc is a function type that implements the Handler interface.
// It provides a convenient way to create handlers without defining new types.
type HandleFunc func(sess *Session) error
