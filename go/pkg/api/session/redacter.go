package session

type Redacter interface {
	// Redact replaces sensitive data and replaces it with `<REDACTED>`
	Redact()
}
