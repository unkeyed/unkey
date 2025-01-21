package session

type Redacter interface {
	// Redact replaces sensitive data and replaces it with `<REDACTED>`
	Redact()
}

type NoopRedacter struct{}

func (r NoopRedacter) Redact() {

}
