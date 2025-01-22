package zen

type Redacter interface {
	// Redact replaces sensitive data with `<REDACTED>`
	// This modifies the underlying struct
	Redact()
}

type NoopRedacter struct{}

func (r NoopRedacter) Redact() {

}
