package invoicecloser

import (
	"context"
	"errors"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
)

// stripeCloser implements Closer against the Stripe API.
type stripeCloser struct {
	client *stripe.Client
}

var _ Closer = (*stripeCloser)(nil)

// NewStripe builds a Stripe-backed Closer from a secret key.
func NewStripe(secretKey string) Closer {
	return &stripeCloser{client: stripe.NewClient(secretKey)}
}

func (c *stripeCloser) ListDraftInvoices(ctx context.Context, stripeSubscriptionID string) ([]DraftInvoice, error) {
	list := c.client.V1Invoices.List(ctx, &stripe.InvoiceListParams{
		ListParams:   stripe.ListParams{Limit: stripe.Int64(100)},
		Subscription: stripe.String(stripeSubscriptionID),
		Status:       stripe.String(string(stripe.InvoiceStatusDraft)),
	})

	var drafts []DraftInvoice
	for invoice, err := range list.All(ctx) {
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to list draft stripe invoices"))
		}
		drafts = append(drafts, DraftInvoice{
			ID:            invoice.ID,
			Status:        string(invoice.Status),
			BillingReason: string(invoice.BillingReason),
			PeriodStart:   invoice.PeriodStart,
			PeriodEnd:     invoice.PeriodEnd,
			AutoAdvance:   invoice.AutoAdvance,
		})
	}
	return drafts, nil
}

// GetInvoice reads one invoice's status, billing reason, and period.
func (c *stripeCloser) GetInvoice(ctx context.Context, invoiceID string) (DraftInvoice, error) {
	invoice, err := c.client.V1Invoices.Retrieve(ctx, invoiceID, nil)
	if err != nil {
		var sErr *stripe.Error
		if errors.As(err, &sErr) && sErr.Code == stripe.ErrorCodeResourceMissing {
			return DraftInvoice{}, ErrNotFound //nolint:exhaustruct // zero value on the not-found path
		}
		return DraftInvoice{}, fault.Wrap(err, fault.Internal("failed to read stripe invoice")) //nolint:exhaustruct // zero value on the error path
	}
	return DraftInvoice{
		ID:            invoice.ID,
		Status:        string(invoice.Status),
		BillingReason: string(invoice.BillingReason),
		PeriodStart:   invoice.PeriodStart,
		PeriodEnd:     invoice.PeriodEnd,
		AutoAdvance:   invoice.AutoAdvance,
	}, nil
}

// ClaimInvoice pushes Stripe's automatic finalization out to finalizeAt, so the
// close can take its time without Stripe closing the invoice underneath it.
//
// auto_advance is set explicitly. Stripe does not infer it from the deadline:
// "If auto_advance isn't already enabled for the invoice, you have to enable it"
// (docs.stripe.com/invoicing/scheduled-finalization). Setting only the deadline
// on an invoice whose auto_advance is false would leave it in draft forever with
// the customer never billed, which is worse than the stale-usage case this
// deadline exists to bound.
func (c *stripeCloser) ClaimInvoice(ctx context.Context, invoiceID string, finalizeAt int64) error {
	_, err := c.client.V1Invoices.Update(ctx, invoiceID, &stripe.InvoiceUpdateParams{ //nolint:exhaustruct // only finalization scheduling changes
		AutoAdvance:              stripe.Bool(true),
		AutomaticallyFinalizesAt: stripe.Int64(finalizeAt),
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to claim stripe invoice"))
	}
	return nil
}

// billed reports whether a status means the invoice was actually finalized and
// charged. draft never went out; void was finalized then reversed (a corrected or
// cancelled invoice), so it represents no charge at all. Mirrors
// billingreconcile.InvoiceStatus.Finalized so the closer and the reconcile engine
// agree on what "billed" means.
func billed(status stripe.InvoiceStatus) bool {
	switch status {
	case stripe.InvoiceStatusOpen, stripe.InvoiceStatusPaid, stripe.InvoiceStatusUncollectible:
		return true
	case stripe.InvoiceStatusDraft, stripe.InvoiceStatusVoid:
		return false
	}
	return false
}

// FinalizeInvoice finalizes a draft. If we lose a race, re-read and return alreadyDone.
//
// A finalize failure on an invoice that is no longer a draft is only "already
// done" when that invoice was genuinely billed. Treating any non-draft status as
// success counted a VOIDED invoice as a closed month: the caller recorded the
// workspace as finalized, the heartbeat stayed green, and nothing was ever
// charged for that period. Void therefore falls through to the error path so the
// workspace is deferred and logged loudly instead.
func (c *stripeCloser) FinalizeInvoice(ctx context.Context, invoiceID string) (bool, error) {
	_, err := c.client.V1Invoices.FinalizeInvoice(ctx, invoiceID, nil)
	if err == nil {
		return false, nil
	}

	invoice, getErr := c.client.V1Invoices.Retrieve(ctx, invoiceID, nil)
	if getErr == nil && billed(invoice.Status) {
		return true, nil
	}
	if getErr == nil && invoice.Status == stripe.InvoiceStatusVoid {
		return false, fault.Wrap(err,
			fault.Internal("stripe invoice was voided, so this period was never billed"))
	}
	return false, fault.Wrap(err, fault.Internal("failed to finalize stripe invoice"))
}
