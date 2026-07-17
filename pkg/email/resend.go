package email

import (
	"net/http"
	"time"
)

// resendEndpoint is Resend's send-email API. Sending with a template is one
// POST here, so a small HTTP client is enough and avoids a dependency.
const resendEndpoint = "https://api.resend.com/emails"

type resendSender struct {
	defaultFrom string
	client      *http.Client
}

// NewResend builds a Resend-backed Sender. defaultFrom is used when an Email
// leaves From empty. The caller decides resend-vs-noop by whether a key is
// configured, so this assumes apiKey is non-empty.
func NewResend(apiKey, defaultFrom string) Sender {
	return &resendSender{
		defaultFrom: defaultFrom,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: authTransport{apiKey: apiKey, next: http.DefaultTransport},
		},
	}
}
