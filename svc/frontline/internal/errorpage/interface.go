package errorpage

// Data contains all the fields available to the error page template.
type Data struct {
	// StatusCode is the HTTP status code (e.g. 401, 502).
	StatusCode int

	// Title is the human-readable status text (e.g. "Unauthorized").
	Title string

	// Message is a longer explanation shown to the user.
	Message string

	// ErrorCode is the URN-style error code (e.g. "err:sentinel:unauthorized:invalid_key").
	ErrorCode string

	// DocsURL links to documentation for this error code. Empty if unavailable.
	DocsURL string

	// RequestID is the frontline request ID for support reference.
	RequestID string
}

// Renderer renders an HTML error page from [Data].
type Renderer interface {
	Render(data Data) ([]byte, error)
}
