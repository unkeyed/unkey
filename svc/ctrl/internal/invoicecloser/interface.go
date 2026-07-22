package invoicecloser

import "context"

// DraftInvoice is what the close flow needs to pick and finalize a renewal draft.
type DraftInvoice struct {
	ID string
	// "subscription_cycle" vs proration invoices the close must skip.
	BillingReason string
	// Invoice period end (unix seconds).
	PeriodEnd int64
	// false means we claimed the draft and are waiting to close it.
	AutoAdvance bool
}

// Closer lists and finalizes draft invoices.
type Closer interface {
	// By subscription so we do not touch another product's drafts.
	ListDraftInvoices(ctx context.Context, stripeSubscriptionID string) ([]DraftInvoice, error)
	// alreadyDone when someone else finalized between list and act.
	FinalizeInvoice(ctx context.Context, invoiceID string) (alreadyDone bool, err error)
}
