// Package stripe registers and routes inbound Stripe webhook events for
// ctrl-api. Transport concerns (signature verification, routing, metrics,
// retry semantics) live in pkg/webhook; each event type is handled in its own
// file, and new events are added as further On(...) registrations in New.
package stripe

import (
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/webhook"
	stripeverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/stripe"
)

// handler holds the dependencies the Stripe event handlers need.
type handler struct {
	restate *restateingress.Client
	stripe  *stripesdk.Client
	db      db.Database
}

// New builds the /webhooks/stripe handler.
func New(
	restateClient *restateingress.Client,
	stripeClient *stripesdk.Client,
	database db.Database,
	webhookSecret string,
) http.Handler {
	h := &handler{
		restate: restateClient,
		stripe:  stripeClient,
		db:      database,
	}
	return webhook.New("stripe", stripeverifier.New(webhookSecret)).
		On([]string{"invoice.created"}, webhook.Typed(h.invoiceCreated))
}
