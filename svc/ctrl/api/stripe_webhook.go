package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	stripesdk "github.com/stripe/stripe-go/v86"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
	stripeverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/stripe"
)

// stripeWebhook holds the dependencies of the Stripe event handlers. The
// transport concerns (signature verification, routing, metrics, retry
// semantics) live in pkg/webhook; this type only contains business logic,
// and new Stripe events are added as further On(...) registrations in
// [NewStripeWebhook].
type stripeWebhook struct {
	restate *restateingress.Client
	stripe  *stripesdk.Client
	db      db.Database
}

// NewStripeWebhook builds the /webhooks/stripe handler.
func NewStripeWebhook(
	restateClient *restateingress.Client,
	stripeClient *stripesdk.Client,
	database db.Database,
	webhookSecret string,
) http.Handler {
	s := &stripeWebhook{
		restate: restateClient,
		stripe:  stripeClient,
		db:      database,
	}
	return webhook.New("stripe", stripeverifier.New(webhookSecret)).
		On(s.handleInvoiceCreated, "invoice.created")
}

// invoiceCreatedPayload is the slice of the invoice.created payload the close
// dispatch needs. Parsed minimally so the webhook tolerates any endpoint API
// version: these fields are stable across Stripe versions.
type invoiceCreatedPayload struct {
	ID            string `json:"id"`
	BillingReason string `json:"billing_reason"`
	Customer      string `json:"customer"`
	// PeriodEnd is the invoice-level period end (unix seconds): the period
	// roll instant for renewal invoices.
	PeriodEnd int64 `json:"period_end"`
}

// handleInvoiceCreated dispatches the month-end Deploy billing close.
//
// On invoice.created for a Deploy workspace's renewal invoice it claims the
// draft (auto_advance off, so Stripe's one-hour auto-finalization stops
// competing with us) and dispatches RunDeployBillingClose for the closed
// period to the worker via Restate. The close itself — final usage push plus
// finalization — happens durably there, not in this handler.
//
// Everything else is ignored: invoices of customers without a Deploy plan
// and proration invoices from subscribes and upgrades keep Stripe's own
// finalization.
func (s *stripeWebhook) handleInvoiceCreated(ctx context.Context, event webhook.Event) error {
	var invoice invoiceCreatedPayload
	if err := json.Unmarshal(event.Payload, &invoice); err != nil {
		return fmt.Errorf("parse invoice.created payload: %w", err)
	}

	// Only renewal invoices close a period; subscribe/upgrade proration
	// invoices (subscription_create / subscription_update) are not ours.
	if invoice.BillingReason != "subscription_cycle" || invoice.Customer == "" || invoice.PeriodEnd == 0 {
		return fmt.Errorf("%w: not a renewal invoice (billing_reason %q)", webhook.ErrIgnore, invoice.BillingReason)
	}

	// Relevance: only workspaces with an active Deploy plan.
	_, err := db.Query.FindDeployWorkspaceByStripeCustomerID(ctx, s.db.RO(), sql.NullString{
		String: invoice.Customer,
		Valid:  true,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fmt.Errorf("%w: customer %s has no deploy workspace", webhook.ErrIgnore, invoice.Customer)
		}
		return fmt.Errorf("workspace lookup for %s: %w", invoice.Customer, err)
	}

	// Claim the draft: without this, Stripe auto-finalizes about an hour
	// after creation and the 01:05 UTC backup cron would find nothing left
	// to correct. Failure is non-fatal — the close usually finalizes within
	// seconds anyway, and auto-finalization is the accepted degraded mode.
	if _, err := s.stripe.V1Invoices.Update(ctx, invoice.ID, &stripesdk.InvoiceUpdateParams{
		AutoAdvance: stripesdk.Bool(false),
	}); err != nil {
		logger.Warn("stripe webhook: failed to disable auto_advance",
			"error", err,
			"invoice_id", invoice.ID,
		)
	}

	// The closed period is the month the invoice's period covered: one
	// second before the roll lands inside it for first cycles (which start
	// mid-month) and full cycles alike.
	period := time.Unix(invoice.PeriodEnd-1, 0).UTC().Format("2006-01")

	client := hydrav1.NewCronServiceIngressClient(s.restate, period)
	_, err = client.RunDeployBillingClose().Send(
		ctx,
		&hydrav1.RunDeployBillingCloseRequest{},
		// One close per period: the per-customer invoice.created storm at the
		// roll dedupes onto a single durable invocation, and Stripe's retries
		// of this event (after a returned error) converge on it too.
		restate.WithIdempotencyKey("deploy-billing-close-"+period),
	)
	if err != nil {
		return fmt.Errorf("dispatch close for %s: %w", period, err)
	}

	logger.Info("stripe webhook: dispatched deploy billing close",
		"billing_period", period,
		"invoice_id", invoice.ID,
	)
	return nil
}
