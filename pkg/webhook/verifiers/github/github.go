// Package github verifies GitHub App webhook signatures for
// pkg/webhook receivers.
package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/webhook"
)

// Verifier checks the X-Hub-Signature-256 header (HMAC-SHA256 over the raw
// body) against the webhook secret. Unlike Stripe, GitHub carries the event
// type and id in headers rather than the payload: X-GitHub-Event routes the
// event and X-GitHub-Delivery identifies it, so the payload is handed to
// handlers verbatim.
type Verifier struct {
	secret string
}

var _ webhook.Verifier = (*Verifier)(nil)

// New builds a Verifier from the GitHub App's webhook secret.
func New(secret string) *Verifier {
	return &Verifier{secret: secret}
}

func (v *Verifier) Verify(r *http.Request, body []byte) (webhook.Event, error) {
	signature := r.Header.Get("X-Hub-Signature-256")
	hexDigest, ok := strings.CutPrefix(signature, "sha256=")
	if !ok {
		return webhook.Event{}, errors.New("missing or malformed X-Hub-Signature-256 header")
	}
	got, err := hex.DecodeString(hexDigest)
	if err != nil {
		return webhook.Event{}, errors.New("malformed X-Hub-Signature-256 hex digest")
	}

	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return webhook.Event{}, errors.New("signature mismatch")
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return webhook.Event{}, errors.New("missing X-GitHub-Event header")
	}

	return webhook.Event{
		ID:      r.Header.Get("X-GitHub-Delivery"),
		Type:    eventType,
		Payload: body,
	}, nil
}
