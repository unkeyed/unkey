package zen

// Handler receives a session and acts on it.
// An error must be returned if anything goes wrong.
//
// The session offers various methods to parse requests and to send
// - JSON
// - Raw bytes
// - Errors
// .
type Handler interface {
	Handle(sess *Session) error
}

type HandleFunc func(sess *Session) error
