package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
)

// resendEndpoint is Resend's send-email API. Sending with a template is one
// POST here, so a small HTTP client is enough and avoids a dependency.
const resendEndpoint = "https://api.resend.com/emails"

type resendSender struct {
	apiKey      string
	defaultFrom string
	client      *http.Client
}

// NewResend builds a Resend-backed Sender. defaultFrom is used when an Email
// leaves From empty. The caller decides resend-vs-noop by whether a key is
// configured, so this assumes apiKey is non-empty.
func NewResend(apiKey, defaultFrom string) Sender {
	return &resendSender{
		apiKey:      apiKey,
		defaultFrom: defaultFrom,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
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
	if len(email.To) == 0 {
		return fault.New("email has no recipients", fault.Internal("email.To is empty"))
	}
	if email.TemplateID == "" {
		return fault.New("email has no template", fault.Internal("email.TemplateID is empty"))
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
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

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
