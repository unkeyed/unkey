package pricing

// Environment identifies a target Stripe account or sandbox. The plan/meter/API
// catalog is identical across environments; only webhook endpoints differ.
type Environment string

const (
	EnvProduction Environment = "production"
	EnvCanary     Environment = "canary"
	EnvSandbox    Environment = "sandbox"
)

// Webhook is a Stripe webhook endpoint. Identity is the URL: reconcile matches
// an existing endpoint by URL rather than creating a duplicate, which would add
// a second signing secret and double-deliver every event.
type Webhook struct {
	Key           string // dashboard | control
	URL           string
	Description   string
	EnabledEvents []string
}

// DashboardWebhookEvents are the events the Next.js dashboard billing handler
// (web/apps/dashboard/app/api/webhooks/stripe/route.ts) acts on. Keep in sync.
var DashboardWebhookEvents = []string{
	"customer.subscription.created",
	"customer.subscription.updated",
	"customer.subscription.deleted",
	"invoice.payment_failed",
	"invoice.payment_succeeded",
}

// Webhooks returns the endpoints desired for env.
//
// Only production has a stable public host. Sandbox and canary have no
// per-account host (previews get generated names), so they declare no endpoint;
// point one at a preview deploy out of band when you need to test there.
//
// The control-plane endpoint is omitted until the Go control-plane Stripe
// handler ships: enabling it earlier makes Stripe retry against a 404 and
// eventually disable the endpoint.
func Webhooks(env Environment) []Webhook {
	switch env {
	case EnvProduction:
		return []Webhook{{
			Key:           "dashboard",
			URL:           "https://app.unkey.com/api/webhooks/stripe",
			Description:   "Unkey dashboard Stripe webhook (managed by unkey-pricing)",
			EnabledEvents: DashboardWebhookEvents,
		}}
	default:
		return nil
	}
}
