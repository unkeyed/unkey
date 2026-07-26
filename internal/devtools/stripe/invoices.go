package stripe

import (
	"context"
	"fmt"

	stripesdk "github.com/stripe/stripe-go/v86"
)

// InvoiceSummary is a dev-tool view of a Stripe invoice.
type InvoiceSummary struct {
	ID               string
	Status           stripesdk.InvoiceStatus
	Total            int64
	Currency         string
	PeriodStart      int64
	PeriodEnd        int64
	HostedInvoiceURL string
	InvoicePDF       string
}

// ListInvoices returns recent invoices for a customer.
func ListInvoices(ctx context.Context, sc *stripesdk.Client, customerID string) ([]InvoiceSummary, error) {
	list := sc.V1Invoices.List(ctx, &stripesdk.InvoiceListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(5)},
		Customer:   stripesdk.String(customerID),
	})
	var invoices []InvoiceSummary
	for invoice, err := range list.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list invoices: %w", err)
		}
		invoices = append(invoices, InvoiceSummary{
			ID:               invoice.ID,
			Status:           invoice.Status,
			Total:            invoice.Total,
			Currency:         string(invoice.Currency),
			PeriodStart:      invoice.PeriodStart,
			PeriodEnd:        invoice.PeriodEnd,
			HostedInvoiceURL: invoice.HostedInvoiceURL,
			InvoicePDF:       invoice.InvoicePDF,
		})
	}
	return invoices, nil
}

// ListInvoicesForClock returns invoices for every customer on a clock.
func ListInvoicesForClock(ctx context.Context, sc *stripesdk.Client, clockID string) (map[string][]InvoiceSummary, error) {
	customers, err := listClockCustomers(ctx, sc, clockID)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]InvoiceSummary, len(customers))
	for _, customer := range customers {
		invoices, err := ListInvoices(ctx, sc, customer.ID)
		if err != nil {
			return nil, err
		}
		out[customer.ID] = invoices
	}
	return out, nil
}
