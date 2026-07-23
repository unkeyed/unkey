package billingreconcile

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// fakeInvoices is an in-memory InvoiceReader. Tests populate candidates (the
// coarse window list, keyed by subscription id) and invoices (full detail,
// keyed by invoice id) directly. The [from, to) window is ignored: fixtures
// scope themselves to exactly the candidates a test wants seen, so re-filtering
// here would just duplicate the Stripe implementation's job.
type fakeInvoices struct {
	candidates map[string][]InvoiceCandidate
	invoices   map[string]Invoice

	findErr error
	getErr  error
}

var _ InvoiceReader = (*fakeInvoices)(nil)

func (f *fakeInvoices) ListInvoices(_ context.Context, subscriptionID string, _, _ time.Time) ([]InvoiceCandidate, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.candidates[subscriptionID], nil
}

func (f *fakeInvoices) GetInvoice(_ context.Context, invoiceID string) (Invoice, error) {
	if f.getErr != nil {
		return Invoice{}, f.getErr //nolint:exhaustruct // fake test double, error path
	}
	inv, ok := f.invoices[invoiceID]
	if !ok {
		return Invoice{}, ErrNotFound //nolint:exhaustruct // fake test double, not-found path
	}
	return inv, nil
}

// fakePrices is an in-memory PriceReader keyed by lookup_key.
type fakePrices struct {
	byLookupKey map[string]Price
	err         error
}

var _ PriceReader = (*fakePrices)(nil)

func (f *fakePrices) PriceByLookupKey(_ context.Context, lookupKey string) (Price, error) {
	if f.err != nil {
		return Price{}, f.err //nolint:exhaustruct // fake test double, error path
	}
	p, ok := f.byLookupKey[lookupKey]
	if !ok {
		return Price{}, ErrNotFound //nolint:exhaustruct // fake test double, not-found path
	}
	return p, nil
}

// fakeUsage is an in-memory UsageReader keyed by workspace id.
type fakeUsage struct {
	byWorkspace map[string]billingmeter.MeterValues
	err         error
}

var _ UsageReader = (*fakeUsage)(nil)

func (f *fakeUsage) WorkspaceUsage(_ context.Context, workspaceID string, _, _ time.Time) (billingmeter.MeterValues, error) {
	if f.err != nil {
		return billingmeter.MeterValues{}, f.err //nolint:exhaustruct // fake test double, error path
	}
	return f.byWorkspace[workspaceID], nil
}
