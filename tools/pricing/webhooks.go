package pricing

import (
	"fmt"
	"net/url"
)

// Environment identifies a target Stripe account or sandbox. The plan/meter/API
// catalog is identical across environments; only webhook endpoints differ.
type Environment string

const (
	EnvProduction Environment = "production"
	EnvCanary     Environment = "canary"
	EnvSandbox    Environment = "sandbox"
)

// Webhook is a Stripe webhook endpoint. Identity is the URL without its query
// string: reconcile matches an existing endpoint by base URL rather than
// creating a duplicate, which would add a second signing secret and
// double-deliver every event. Query parameters (the Vercel bypass secret) are
// updated in place on the matched endpoint, which keeps its signing secret.
type Webhook struct {
	Key           string // dashboard | control
	URL           string // base URL, no query string
	Description   string
	EnabledEvents []string
	// VercelProtected marks a host behind Vercel deployment protection, which
	// rejects requests before they reach the handler. Stripe cannot send custom
	// headers, so delivery needs the protection-bypass secret as a query
	// parameter (see DeliveryURL).
	VercelProtected bool
}

// VercelBypassParam is the query parameter Vercel accepts its "Protection
// Bypass for Automation" secret on, for callers that cannot set headers.
const VercelBypassParam = "x-vercel-protection-bypass"

// DeliveryURL is the URL Stripe delivers to: the base URL, plus the bypass
// secret for Vercel-protected hosts. The secret is deployment-local
// configuration (VERCEL_PROTECTION_BYPASS_<ENV> in tools/pricing/.env), never
// source. A protected endpoint without a secret is an error rather than a
// fallback to the bare URL, so an apply can never silently strip the parameter
// off a live endpoint and break delivery.
func (w Webhook) DeliveryURL(vercelBypassSecret string) (string, error) {
	if !w.VercelProtected {
		return w.URL, nil
	}
	if vercelBypassSecret == "" {
		return "", fmt.Errorf("%s is behind Vercel deployment protection: set VERCEL_PROTECTION_BYPASS_<ENV> in tools/pricing/.env (see .env.example)", w.URL)
	}
	return w.URL + "?" + VercelBypassParam + "=" + url.QueryEscape(vercelBypassSecret), nil
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

// ControlWebhookEvents are the events the Go control-plane billing handler
// (svc/ctrl/api/webhooks/stripe/stripe.go) acts on. Keep in sync.
var ControlWebhookEvents = []string{
	"invoice.created",
}

// Webhooks returns the endpoints desired for env.
//
// Production and canary have stable public hosts; sandbox has no per-account
// host (previews get generated names), so it declares no endpoint — point one
// at a preview deploy out of band when you need to test there.
//
// The control-plane handler serves /webhooks/stripe (no /api prefix, unlike
// the dashboard) and only registers the route once its stripe secrets are
// configured. Don't apply a control endpoint before the handler is live at
// that host: Stripe retries against the 404 and eventually disables the
// endpoint.
func Webhooks(env Environment) []Webhook {
	switch env {
	case EnvProduction:
		return []Webhook{{
			Key:           "dashboard",
			URL:           "https://app.unkey.com/api/webhooks/stripe",
			Description:   "Unkey dashboard Stripe webhook (managed by unkey-pricing)",
			EnabledEvents: DashboardWebhookEvents,
		}, {
			Key:           "control",
			URL:           "https://control.unkey.cloud/webhooks/stripe",
			Description:   "Unkey control-plane Stripe webhook (managed by unkey-pricing)",
			EnabledEvents: ControlWebhookEvents,
		}}
	case EnvCanary:
		return []Webhook{{
			Key:             "dashboard",
			URL:             "https://app.unkey-canary.com/api/webhooks/stripe",
			Description:     "Unkey canary dashboard Stripe webhook (managed by unkey-pricing)",
			EnabledEvents:   DashboardWebhookEvents,
			VercelProtected: true,
		}, {
			Key:           "control",
			URL:           "https://control.unkey-canary.com/webhooks/stripe",
			Description:   "Unkey canary control-plane Stripe webhook (managed by unkey-pricing)",
			EnabledEvents: ControlWebhookEvents,
		}}
	default:
		return nil
	}
}
