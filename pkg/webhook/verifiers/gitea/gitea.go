// Package gitea verifies Gitea webhook signatures for pkg/webhook receivers.
package gitea

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/webhook"
)

// Verifier checks the X-Gitea-Signature header (hex HMAC-SHA256 over the raw
// body, no scheme prefix) against the webhook secret. The event type travels
// in X-Gitea-Event (e.g. "push") and the delivery id in X-Gitea-Delivery.
// Gitea also sends X-GitHub-* compatibility headers; the native ones are used
// so a future GitHub header change cannot silently alter Gitea verification.
type Verifier struct {
	secret string
}

var _ webhook.Verifier = (*Verifier)(nil)

// New builds a Verifier from the webhook's secret.
func New(secret string) *Verifier {
	return &Verifier{secret: secret}
}

func (v *Verifier) Verify(r *http.Request) (webhook.Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return webhook.Event{}, err
	}

	signature := r.Header.Get("X-Gitea-Signature")
	if signature == "" {
		return webhook.Event{}, errors.New("missing X-Gitea-Signature header")
	}
	got, err := hex.DecodeString(signature)
	if err != nil {
		return webhook.Event{}, errors.New("malformed X-Gitea-Signature hex digest")
	}

	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return webhook.Event{}, errors.New("signature mismatch")
	}

	eventType := r.Header.Get("X-Gitea-Event")
	if eventType == "" {
		return webhook.Event{}, errors.New("missing X-Gitea-Event header")
	}

	// X-Gitea-Delivery is Event.ID, the idempotency key for downstream
	// dispatch. Reject an empty one rather than silently disabling dedup.
	deliveryID := r.Header.Get("X-Gitea-Delivery")
	if deliveryID == "" {
		return webhook.Event{}, errors.New("missing X-Gitea-Delivery header")
	}

	return webhook.Event{
		ID:      deliveryID,
		Type:    eventType,
		Payload: body,
	}, nil
}
