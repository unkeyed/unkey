package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (s *resendSender) Send(ctx context.Context, email Email) error {
	err := assert.All(
		assert.NotEmpty(email.To, "email has no recipients"),
		assert.NotEmpty(email.TemplateID, "email has no template"),
		validateIdempotencyKey(email.IdempotencyKey),
	)
	if err != nil {
		return fault.Wrap(err, fault.Internal("invalid email"))
	}

	from := email.From
	if from == "" {
		from = s.defaultFrom
	}

	body, err := json.Marshal(newResendRequest(from, email))
	if err != nil {
		return fault.Wrap(err, fault.Internal("marshal resend request"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEndpoint, bytes.NewReader(body))
	if err != nil {
		return fault.Wrap(err, fault.Internal("build resend request"))
	}
	if email.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", email.IdempotencyKey)
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

func validateIdempotencyKey(key string) error {
	if key == "" {
		return nil
	}
	return assert.InRange(len(key), 1, 256, "idempotency key length")
}
