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

// deploySubscriptionID returns the subscription that generated this invoice.
//
// Read through the SDK's own types rather than a hand-rolled json tag. Stripe
// moved this from a top-level `subscription` field to
// `parent.subscription_details.subscription` in API version 2025-03-31.basil, and
// a plain string tag decodes the new shape to "" without erroring, which silently
// turned every renewal into an ignored event. Binding to the generated type makes
// the next such move a compile failure on SDK upgrade instead of a silent empty
// string.
//
// Each level is nil-checked because Stripe only populates `parent` for invoices
// that have one, and only fills `subscription_details` when the parent is a
// subscription. The SDK's Subscription unmarshals an unexpanded string id
// straight into ID, so this needs no expand.
func deploySubscriptionID(invoice stripesdk.Invoice) string {
	if invoice.Parent == nil ||
		invoice.Parent.SubscriptionDetails == nil ||
		invoice.Parent.SubscriptionDetails.Subscription == nil {
		return ""
	}
	return invoice.Parent.SubscriptionDetails.Subscription.ID
}

// customerID returns the invoice's customer id, or "" when absent. Customer is
// an expandable reference, so the unexpanded webhook payload carries only the id.
func customerID(invoice stripesdk.Invoice) string {
	if invoice.Customer == nil {
		return ""
	}
	return invoice.Customer.ID
}

// invoiceCreated closes the Deploy renewal invoice Stripe just created.
//
// The payload is decoded as the SDK's stripesdk.Invoice, which is generated from
// Stripe's OpenAPI spec, so the field set tracks the API version the SDK targets
// and there are no local json tags to drift out of date.
func (h *handler) invoiceCreated(
	ctx context.Context,
	_ webhook.Event,
	invoice stripesdk.Invoice,
) error {
	subscriptionID := deploySubscriptionID(invoice)
	customer := customerID(invoice)

	// Not a renewal invoice. Manual and custom invoices are left to Stripe.
	if invoice.BillingReason != stripesdk.InvoiceBillingReasonSubscriptionCycle ||
		customer == "" ||
		subscriptionID == "" ||
		invoice.PeriodStart == 0 ||
		invoice.PeriodEnd == 0 {
		return fmt.Errorf("%w: not a renewal invoice (billing_reason %q, subscription %q)",
			webhook.ErrIgnore, invoice.BillingReason, subscriptionID)
	}

	// Deploy workspace only: customers without deploy_plan are ignored and
	// Stripe keeps auto-finalizing on its own schedule.
	ws, err := h.db.FindDeployWorkspaceByStripeCustomerID(ctx, sql.NullString{
		String: customer,
		Valid:  true,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fmt.Errorf("%w: customer %s has no deploy workspace", webhook.ErrIgnore, customer)
		}
		return fmt.Errorf("workspace lookup for %s: %w", customer, err)
	}

	// Same customer can hold multiple Stripe subscriptions. Only claim and
	// close renewals for this workspace's Deploy subscription.
	if !ws.StripeDeploySubscriptionID.Valid || ws.StripeDeploySubscriptionID.String == "" {
		return fmt.Errorf("%w: deploy workspace %s has no stripe subscription id", webhook.ErrIgnore, ws.ID)
	}
	if subscriptionID != ws.StripeDeploySubscriptionID.String {
		return fmt.Errorf("%w: invoice subscription %s is not workspace deploy subscription %s",
			webhook.ErrIgnore, subscriptionID, ws.StripeDeploySubscriptionID.String)
	}

	// Replace Stripe's ~1h auto-finalization with a scheduled one at period end
	// plus 48 hours. That holds the draft open through the close's 24h usage
	// ingest wait, but unlike a bare auto_advance=false it bounds the failure
	// mode where the close never runs: the invoice then finalizes with the last
	// hourly push's usage (at most an hour stale, still inside the credit
	// expiry window) instead of sitting open forever with the customer never
	// billed. The close's manual finalize normally wins the race; the schedule
	// only fires on a draft. Webhook must succeed or Stripe redelivers.
	//
	// auto_advance is set explicitly alongside the deadline. Stripe does NOT
	// infer it: "If auto_advance isn't already enabled for the invoice, you have
	// to enable it" (docs.stripe.com/invoicing/scheduled-finalization). Without
	// it, an invoice that arrived with auto_advance false would take the schedule
	// and then never act on it, sitting in draft forever with the customer never
	// billed, which is the exact failure this backstop exists to prevent.
	backstopAt := invoice.PeriodEnd + int64((48 * time.Hour).Seconds())
	if _, err := h.stripe.V1Invoices.Update(ctx, invoice.ID, &stripesdk.InvoiceUpdateParams{
		AutoAdvance:              stripesdk.Bool(true),
		AutomaticallyFinalizesAt: stripesdk.Int64(backstopAt),
	}); err != nil {
		// automatically_finalizes_at is draft-only. A redelivery arriving after
		// the invoice finalized must not error forever; the close path already
		// handles finalized invoices, so skip the claim and continue.
		live, getErr := h.stripe.V1Invoices.Retrieve(ctx, invoice.ID, nil)
		if getErr != nil || live.Status == stripesdk.InvoiceStatusDraft {
			return fmt.Errorf("schedule backstop finalization on invoice %s: %w", invoice.ID, err)
		}
		logger.Info("invoice already finalized, skipping backstop claim",
			"invoice_id", invoice.ID, "status", live.Status)
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
