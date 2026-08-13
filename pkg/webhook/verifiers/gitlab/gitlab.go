// Package gitlab verifies GitLab webhook tokens for pkg/webhook receivers.
package gitlab

import (
	"crypto/subtle"
	"errors"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/webhook"
)

// Verifier checks the X-Gitlab-Token header against the configured secret.
// GitLab does not sign payloads: the secret token is sent verbatim with every
// delivery, so verification is a constant-time equality check. Like GitHub,
// the event type and id travel in headers (X-Gitlab-Event,
// X-Gitlab-Event-UUID) and the payload is handed to handlers verbatim.
type Verifier struct {
	secret string
}

var _ webhook.Verifier = (*Verifier)(nil)

// New builds a Verifier from the webhook's secret token.
func New(secret string) *Verifier {
	return &Verifier{secret: secret}
}

func (v *Verifier) Verify(r *http.Request) (webhook.Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return webhook.Event{}, err
	}

	token := r.Header.Get("X-Gitlab-Token")
	if token == "" {
		return webhook.Event{}, errors.New("missing X-Gitlab-Token header")
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(v.secret)) != 1 {
		return webhook.Event{}, errors.New("token mismatch")
	}

	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType == "" {
		return webhook.Event{}, errors.New("missing X-Gitlab-Event header")
	}

	// X-Gitlab-Event-UUID is Event.ID, the idempotency key for downstream
	// dispatch. Reject an empty one rather than silently disabling dedup.
	eventUUID := r.Header.Get("X-Gitlab-Event-UUID")
	if eventUUID == "" {
		return webhook.Event{}, errors.New("missing X-Gitlab-Event-UUID header")
	}

	return webhook.Event{
		ID:      eventUUID,
		Type:    eventType,
		Payload: body,
	}, nil
}
