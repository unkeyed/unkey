package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
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

// resendTemplate is the nested template object in the send payload. Resend
// rejects a payload that mixes a template with html/text, so this is the only
// content carrier.
type resendTemplate struct {
	ID        string            `json:"id"`
	Variables map[string]string `json:"variables,omitempty"`
}

type resendRequest struct {
	// From and Subject are omitempty so a template-only send can leave them out
	// and let the published template's own From and Subject apply. Setting either
	// overrides the template.
	From     string         `json:"from,omitempty"`
	To       []string       `json:"to"`
	Subject  string         `json:"subject,omitempty"`
	Template resendTemplate `json:"template"`
}

func (s *resendSender) Send(ctx context.Context, email Email) error {
	err := assert.All(
		assert.NotEmpty(email.To, "email has no recipients"),
		assert.NotEmpty(email.TemplateID, "email has no template"),
	)
	if err != nil {
		return fault.Wrap(err, fault.Internal("invalid email"))
	}

	from := email.From
	if from == "" {
		from = s.defaultFrom
	}

	body, err := json.Marshal(resendRequest{
		From:     from,
		To:       email.To,
		Subject:  email.Subject,
		Template: resendTemplate{ID: email.TemplateID, Variables: email.Variables},
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("marshal resend request"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEndpoint, bytes.NewReader(body))
	if err != nil {
		return fault.Wrap(err, fault.Internal("build resend request"))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fault.Wrap(err, fault.Internal("send resend request"))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusMultipleChoices {
		// Cap the body: provider errors are small JSON, not payloads.
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fault.New(
			fmt.Sprintf("resend returned %d", resp.StatusCode),
			fault.Internal(fmt.Sprintf("resend send failed (%d): %s", resp.StatusCode, respBody)),
		)
	}
	return nil
}
