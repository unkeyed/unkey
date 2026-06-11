package invoicecloser

import "context"

// noopCloser sees no invoices and finalizes nothing. Used when Stripe is not
// configured so the close handler can run unconditionally.
type noopCloser struct{}

var _ Closer = (*noopCloser)(nil)

// NewNoop returns a Closer that reports no drafts.
func NewNoop() Closer { return &noopCloser{} }

func (n *noopCloser) ListDraftInvoices(_ context.Context, _ string) ([]DraftInvoice, error) {
	return nil, nil
}

func (n *noopCloser) FinalizeInvoice(_ context.Context, _ string) (bool, error) {
	return true, nil
}
