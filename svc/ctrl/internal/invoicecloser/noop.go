package invoicecloser

import "context"

// noopCloser for when Stripe is not configured.
type noopCloser struct{}

var _ Closer = (*noopCloser)(nil)

// NewNoop returns a Closer that reports no drafts.
func NewNoop() Closer { return &noopCloser{} }

func (n *noopCloser) ListDraftInvoices(_ context.Context, _ string) ([]DraftInvoice, error) {
	return nil, nil
}

func (n *noopCloser) GetInvoice(_ context.Context, _ string) (DraftInvoice, error) {
	return DraftInvoice{}, ErrNotFound //nolint:exhaustruct // nothing exists when Stripe is not configured
}

func (n *noopCloser) ClaimInvoice(_ context.Context, _ string, _ int64) error {
	return nil
}

func (n *noopCloser) FinalizeInvoice(_ context.Context, _ string) (bool, error) {
	return true, nil
}
