// Package bitbucket verifies Bitbucket Cloud webhook signatures for
// pkg/webhook receivers.
package bitbucket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/webhook"
)

// Verifier checks the X-Hub-Signature header (HMAC-SHA256 over the raw body)
// against the webhook secret. Bitbucket signs like GitHub but uses the legacy
// header name without the -256 suffix. The event type travels in X-Event-Key
// (e.g. "repo:push") and the delivery id in X-Request-UUID; X-Hook-UUID is the
// hook's identity, identical on every delivery, so it must not be used for
// dedup.
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

	signature := r.Header.Get("X-Hub-Signature")
	hexDigest, ok := strings.CutPrefix(signature, "sha256=")
	if !ok {
		return webhook.Event{}, errors.New("missing or malformed X-Hub-Signature header")
	}
	got, err := hex.DecodeString(hexDigest)
	if err != nil {
		return webhook.Event{}, errors.New("malformed X-Hub-Signature hex digest")
	}

	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return webhook.Event{}, errors.New("signature mismatch")
	}

	eventType := r.Header.Get("X-Event-Key")
	if eventType == "" {
		return webhook.Event{}, errors.New("missing X-Event-Key header")
	}

	// X-Request-UUID is Event.ID, the idempotency key for downstream dispatch.
	// Reject an empty one rather than silently disabling dedup.
	requestUUID := r.Header.Get("X-Request-UUID")
	if requestUUID == "" {
		return webhook.Event{}, errors.New("missing X-Request-UUID header")
	}

	return webhook.Event{
		ID:      requestUUID,
		Type:    eventType,
		Payload: body,
	}, nil
}
