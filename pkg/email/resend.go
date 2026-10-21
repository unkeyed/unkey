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

// NewResend returns a Resend-backed Sender, or a noop when apiKey is empty.
// defaultFrom is used when an Email leaves From empty.
func NewResend(apiKey, defaultFrom string) Sender {
	if apiKey == "" {
		return NewNoop()
	}
	return &resendSender{
		defaultFrom: defaultFrom,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: authTransport{apiKey: apiKey, next: http.DefaultTransport},
		},
	}
}
