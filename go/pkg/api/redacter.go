package api

type Redacter[T any] interface {
	// Redact replaces sensitive data and replaces it with `<REDACTED>`
	Redact() T
}
