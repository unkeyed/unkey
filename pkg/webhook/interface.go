package webhook

import (
	"context"
	"errors"
	"net/http"
)

// ErrIgnore marks an event a handler examined and deliberately declined.
// Wrap it to carry the reason into logs:
//
//	return fmt.Errorf("%w: not a deploy workspace", webhook.ErrIgnore)
var ErrIgnore = errors.New("webhook event ignored")

// Event is a verified provider event: identity, type, and the raw payload
// for the handler to parse into its provider-specific shape.
type Event struct {
	// ID is the provider-assigned event id, for logs and idempotency.
	ID string
	// Type routes the event, e.g. "invoice.created".
	Type string
	// Payload is the raw event object (for Stripe, event.data.object).
	Payload []byte
}

// Verifier authenticates a request and extracts its event. Implementations
// are provider-specific (signature schemes differ); the returned error is
// never exposed to the caller, only logged.
type Verifier interface {
	Verify(r *http.Request, body []byte) (Event, error)
}

// HandlerFunc processes one verified event. See the package documentation
// for how the returned error maps to HTTP responses and retries.
type HandlerFunc func(ctx context.Context, event Event) error

// Middleware wraps every registered handler, outermost first. Use it for
// integration-specific concerns the transport cannot know about: custom
// metrics, tracing attributes, payload archival.
type Middleware func(next HandlerFunc) HandlerFunc
