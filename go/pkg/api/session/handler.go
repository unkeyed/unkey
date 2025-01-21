package session

// Handler receives a session and acts on it.
// An error must be returned if anything goes wrong.
//
// The session offers various methods to parse requests and to send
// - JSON
// - Raw bytes
// - Errors
type Handler[TRequest Redacter, TResponse Redacter] interface {
	Handle(sess Session[TRequest, TResponse]) error
}
