package email

import "net/http"

// authTransport sets the Resend credentials and content type on every request
// going through the client, so no call site can forget them.
type authTransport struct {
	apiKey string
	next   http.RoundTripper
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// RoundTrippers must not mutate the caller's request.
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return t.next.RoundTrip(req)
}
