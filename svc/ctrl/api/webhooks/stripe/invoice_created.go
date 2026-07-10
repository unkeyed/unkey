package stripe

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	stripesdk "github.com/stripe/stripe-go/v86"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// invoiceCreatedPayload is the invoice.created fields the close path needs.
// Kept minimal so any Stripe API version works.
type invoiceCreatedPayload struct {
	ID            string `json:"id"`
	BillingReason string `json:"billing_reason"`
	Customer      string `json:"customer"`
	// Deploy subscription that generated this renewal invoice.
	Subscription string `json:"subscription"`
	// Billed period bounds (unix seconds).
	PeriodStart int64 `json:"period_start"`
	PeriodEnd   int64 `json:"period_end"`
}

func (h *handler) invoiceCreated(
	ctx context.Context,
	_ webhook.Event,
	invoice invoiceCreatedPayload,
) error {
	// Not a renewal invoice. Manual and custom invoices are left to Stripe.
	if invoice.BillingReason != "subscription_cycle" ||
		invoice.Customer == "" ||
		invoice.Subscription == "" ||
		invoice.PeriodStart == 0 ||
		invoice.PeriodEnd == 0 {
		return fmt.Errorf("%w: not a renewal invoice (billing_reason %q)", webhook.ErrIgnore, invoice.BillingReason)
	}

	// Deploy workspace only: customers without deploy_plan are ignored and
	// Stripe keeps auto-finalizing on its own schedule.
	ws, err := h.db.FindDeployWorkspaceByStripeCustomerID(ctx, sql.NullString{
		String: invoice.Customer,
		Valid:  true,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fmt.Errorf("%w: customer %s has no deploy workspace", webhook.ErrIgnore, invoice.Customer)
		}
		return fmt.Errorf("workspace lookup for %s: %w", invoice.Customer, err)
	}

	// Same customer can hold multiple Stripe subscriptions. Only claim and
	// close renewals for this workspace's Deploy subscription.
	if !ws.StripeSubscriptionID.Valid || ws.StripeSubscriptionID.String == "" {
		return fmt.Errorf("%w: deploy workspace %s has no stripe subscription id", webhook.ErrIgnore, ws.ID)
	}
	if invoice.Subscription != ws.StripeSubscriptionID.String {
		return fmt.Errorf("%w: invoice subscription %s is not workspace deploy subscription %s",
			webhook.ErrIgnore, invoice.Subscription, ws.StripeSubscriptionID.String)
	}

	// Stop Stripe auto-finalizing ~1h later with stale usage. Webhook must
	// succeed or Stripe redelivers.
	if _, err := h.stripe.V1Invoices.Update(ctx, invoice.ID, &stripesdk.InvoiceUpdateParams{
		AutoAdvance: stripesdk.Bool(false),
	}); err != nil {
		return fmt.Errorf("disable auto_advance on invoice %s: %w", invoice.ID, err)
	}

	// Closed month from period_start: always inside the billed period, unlike
	// period_end-1s which drifts when the anchor is not exactly midnight UTC.
	period := time.Unix(invoice.PeriodStart, 0).UTC().Format("2006-01")

	client := hydrav1.NewCronServiceIngressClient(h.restate, ws.ID)
	_, err = client.CloseDeployBillingWorkspace().Send(
		ctx,
		&hydrav1.CloseDeployBillingWorkspaceRequest{
			Period:    period,
			PeriodEnd: invoice.PeriodEnd,
			InvoiceId: invoice.ID,
		},
		restate.WithIdempotencyKey("deploy-billing-close-"+period+"-"+invoice.ID),
	)
	if err != nil {
		return fmt.Errorf("dispatch close for workspace %s: %w", ws.ID, err)
	}

	logger.Info("stripe webhook: dispatched deploy billing close",
		"workspace_id", ws.ID,
		"billing_period", period,
		"invoice_id", invoice.ID,
	)
	return nil
}
