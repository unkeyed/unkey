package invoicecloser

import (
	"context"
	"errors"
)

// DraftInvoice is what the close flow needs to pick and finalize a renewal draft.
type DraftInvoice struct {
	ID string
	// Stripe invoice lifecycle status (draft, open, paid, void, or
	// uncollectible). GetInvoice can return any status; ListDraftInvoices only
	// returns draft invoices.
	Status string
	// "subscription_cycle" vs proration invoices the close must skip.
	BillingReason string
	// Invoice period start (unix seconds). Asserted against the calendar-month
	// start at close so an anchor-misaligned invoice never finalizes.
	PeriodStart int64
	// Invoice period end (unix seconds).
	PeriodEnd int64
	// false means we claimed the draft and are waiting to close it.
	AutoAdvance bool
}

// Closer lists and finalizes draft invoices.
type Closer interface {
	// By subscription so we do not touch another product's drafts.
	ListDraftInvoices(ctx context.Context, stripeSubscriptionID string) ([]DraftInvoice, error)
	// GetInvoice reads one invoice's status, billing reason, and period. The
	// webhook close path is handed an invoice id rather than discovering it by
	// listing, so it needs this to detect redelivery after finalization and run
	// the same period-alignment assertion the fleet sweep runs before finalizing.
	// Returns ErrNotFound when the invoice does not exist.
	GetInvoice(ctx context.Context, invoiceID string) (DraftInvoice, error)
	// ClaimInvoice pushes out Stripe's automatic finalization to finalizeAt (unix
	// seconds), holding the draft open so the close can take its time. Without a
	// claim Stripe auto-finalizes roughly an hour after creating the invoice, which
	// is well inside the close's 24h usage-ingestion wait. The invoice.created
	// webhook claims per invoice; the fleet sweep claims whatever the webhook
	// missed. Idempotent: setting the same deadline twice is a no-op.
	ClaimInvoice(ctx context.Context, invoiceID string, finalizeAt int64) error
	// alreadyDone when someone else finalized between list and act.
	FinalizeInvoice(ctx context.Context, invoiceID string) (alreadyDone bool, err error)
}

// ErrNotFound is returned by GetInvoice when Stripe has no such invoice, as
// distinct from a transient read failure. The caller defers on either, but only
// this one is terminal.
var ErrNotFound = errors.New("invoicecloser: invoice not found")
