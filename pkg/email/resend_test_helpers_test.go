package email

import "net/http"

// newTestResend points the sender at a stub server instead of api.resend.com,
// keeping the sender's own transport (and its header injection) in the chain.
func newTestResend(url, apiKey, from string) *resendSender {
	s := NewResend(apiKey, from).(*resendSender)
	s.client.Transport = rewriteHost{base: url, next: s.client.Transport}
	return s
}

// rewriteHost sends every request to the stub server, regardless of the
// hardcoded resend endpoint.
type rewriteHost struct {
	base string
	next http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	stub, err := http.NewRequestWithContext(req.Context(), req.Method, r.base, req.Body)
	if err != nil {
		return nil, err
	}
	stub.Header = req.Header
	return r.next.RoundTrip(stub)
}
