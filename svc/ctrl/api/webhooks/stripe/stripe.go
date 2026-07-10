// Package stripe routes Stripe webhooks for ctrl-api.
// Transport lives in pkg/webhook; add events as On(...) in New.
package stripe

import (
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/webhook"
	stripeverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/stripe"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

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
