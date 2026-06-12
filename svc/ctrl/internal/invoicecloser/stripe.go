package invoicecloser

import (
	"context"

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

func (c *stripeCloser) ListDraftInvoices(ctx context.Context, stripeCustomerID string) ([]DraftInvoice, error) {
	list := c.client.V1Invoices.List(ctx, &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(10)},
		Customer:   stripe.String(stripeCustomerID),
		Status:     stripe.String(string(stripe.InvoiceStatusDraft)),
	})

	var drafts []DraftInvoice
	for invoice, err := range list.All(ctx) {
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to list draft stripe invoices"))
		}
		drafts = append(drafts, DraftInvoice{
			ID:            invoice.ID,
			BillingReason: string(invoice.BillingReason),
			PeriodEnd:     invoice.PeriodEnd,
		})
	}
	return drafts, nil
}

// FinalizeInvoice finalizes a draft. Losing a race (the invoice was
// finalized between listing and acting, e.g. by the webhook path while the
// backup cron sweeps) is success, not failure: rather than pattern-matching
// Stripe error codes, a failed finalize re-reads the invoice and reports
// alreadyDone when it is no longer a draft.
func (c *stripeCloser) FinalizeInvoice(ctx context.Context, invoiceID string) (bool, error) {
	_, err := c.client.V1Invoices.FinalizeInvoice(ctx, invoiceID, nil)
	if err == nil {
		return false, nil
	}

	invoice, getErr := c.client.V1Invoices.Retrieve(ctx, invoiceID, nil)
	if getErr == nil && invoice.Status != stripe.InvoiceStatusDraft {
		return true, nil
	}
	return false, fault.Wrap(err, fault.Internal("failed to finalize stripe invoice"))
}
