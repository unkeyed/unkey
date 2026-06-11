package invoicecloser

import "context"

// DraftInvoice is a draft invoice as the close flow sees it: enough to decide
// whether it is a renewal invoice for the closed period and to finalize it.
type DraftInvoice struct {
	ID string
	// BillingReason distinguishes renewal invoices ("subscription_cycle")
	// from subscribe/upgrade proration invoices, which the close must not
	// touch.
	BillingReason string
	// PeriodEnd is the invoice-level period end (unix seconds).
	PeriodEnd int64
}

// Closer lists and finalizes draft invoices. Implemented against Stripe;
// faked in tests and disabled with the noop.
type Closer interface {
	// ListDraftInvoices returns the subscription's draft invoices. Scoping to
	// the workspace's own subscription (not the customer) keeps the close away
	// from drafts of any other subscription the customer might have.
	ListDraftInvoices(ctx context.Context, stripeSubscriptionID string) ([]DraftInvoice, error)
	// FinalizeInvoice finalizes a draft invoice. Returns alreadyDone=true
	// when the invoice is no longer a draft (someone else finalized it), so
	// callers can treat replays as success.
	FinalizeInvoice(ctx context.Context, invoiceID string) (alreadyDone bool, err error)
}
