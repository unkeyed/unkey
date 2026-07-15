// Package stripe verifies Stripe webhook signatures for
// pkg/webhook receivers.
package stripe

import (
	"io"
	"net/http"

	stripewebhook "github.com/stripe/stripe-go/v86/webhook"
	"github.com/unkeyed/unkey/pkg/webhook"
)

// Verifier checks the Stripe-Signature header against the endpoint's signing
// secret and unwraps the event. The payload handed to handlers is the event's
// data.object (e.g. the invoice itself), not the event envelope.
type Verifier struct {
	secret string
}

var _ webhook.Verifier = (*Verifier)(nil)

// New builds a Verifier from the endpoint's signing secret (whsec_...).
func New(secret string) *Verifier {
	return &Verifier{secret: secret}
}

func (v *Verifier) Verify(r *http.Request) (webhook.Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return webhook.Event{}, err
	}

	// IgnoreAPIVersionMismatch: stripe-go otherwise rejects events whose
	// api_version differs from the SDK's pin, turning an endpoint-version
	// drift into a total webhook outage. Verification stays strict on the
	// signature; handlers are expected to parse the raw payload minimally,
	// reading only fields that are stable across Stripe API versions.
	//nolint:exhaustruct // defaults are correct for everything else
	event, err := stripewebhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		v.secret,
		stripewebhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		return webhook.Event{}, err
	}

	return webhook.Event{
		ID:      event.ID,
		Type:    string(event.Type),
		Payload: event.Data.Raw,
	}, nil
}
