package stripe

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	stripesdk "github.com/stripe/stripe-go/v86"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
)

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

// invoiceCreated dispatches the month-end Deploy billing close. For a Deploy
// workspace's renewal invoice it disables auto_advance (so Stripe won't
// auto-finalize before the final usage is pushed) and hands off to
// RunDeployBillingClose on the worker. Non-renewal invoices and non-Deploy
// customers are ignored.
func (h *handler) invoiceCreated(
	ctx context.Context,
	_ webhook.Event,
	invoice invoiceCreatedPayload,
) error {
	// Only renewal invoices close a period; subscribe/upgrade proration
	// invoices (subscription_create / subscription_update) are not ours.
	if invoice.BillingReason != "subscription_cycle" || invoice.Customer == "" || invoice.PeriodEnd == 0 {
		return fmt.Errorf("%w: not a renewal invoice (billing_reason %q)", webhook.ErrIgnore, invoice.BillingReason)
	}

	// Relevance: only workspaces with an active Deploy plan.
	_, err := db.Query.FindDeployWorkspaceByStripeCustomerID(ctx, h.db.RO(), sql.NullString{
		String: invoice.Customer,
		Valid:  true,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fmt.Errorf("%w: customer %s has no deploy workspace", webhook.ErrIgnore, invoice.Customer)
		}
		return fmt.Errorf("workspace lookup for %s: %w", invoice.Customer, err)
	}

	// Disable auto_advance: otherwise Stripe finalizes the draft ~1h after
	// creation, before the backup cron can push the final total. Fatal: if
	// this update is lost and the close's own finalize also fails, Stripe
	// auto-finalizes with stale usage, which is the exact outcome the close
	// exists to prevent. Failing the webhook makes Stripe redeliver it.
	if _, err := h.stripe.V1Invoices.Update(ctx, invoice.ID, &stripesdk.InvoiceUpdateParams{
		AutoAdvance: stripesdk.Bool(false),
	}); err != nil {
		return fmt.Errorf("disable auto_advance on invoice %s: %w", invoice.ID, err)
	}

	// The closed period is the month the invoice's period covered: one
	// second before the roll lands inside it for first cycles (which start
	// mid-month) and full cycles alike.
	period := time.Unix(invoice.PeriodEnd-1, 0).UTC().Format("2006-01")

	client := hydrav1.NewCronServiceIngressClient(h.restate, period)
	_, err = client.RunDeployBillingClose().Send(
		ctx,
		&hydrav1.RunDeployBillingCloseRequest{},
		// One close per period: the invoice.created storm at the roll dedupes
		// onto a single invocation. The backup cron uses a distinct key, so it
		// is a real retry, not a replay of this one.
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
