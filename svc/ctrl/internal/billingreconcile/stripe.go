package billingreconcile

import (
	"context"
	"time"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
)

// stripeInvoiceReader implements InvoiceReader against the Stripe API.
type stripeInvoiceReader struct {
	client *stripe.Client
}

var _ InvoiceReader = (*stripeInvoiceReader)(nil)

// NewStripeInvoiceReader builds a Stripe-backed InvoiceReader from a secret key.
func NewStripeInvoiceReader(secretKey string) InvoiceReader {
	return &stripeInvoiceReader{client: stripe.NewClient(secretKey)}
}

func (r *stripeInvoiceReader) ListInvoices(
	ctx context.Context,
	subscriptionID string,
	from, to time.Time,
) ([]InvoiceCandidate, error) {
	list := r.client.V1Invoices.List(ctx, &stripe.InvoiceListParams{
		ListParams:   stripe.ListParams{Limit: stripe.Int64(100)}, //nolint:exhaustruct // only Limit is set; the rest default
		Subscription: stripe.String(subscriptionID),
		CreatedRange: &stripe.RangeQueryParams{
			GreaterThanOrEqual: from.Unix(),
			LesserThanOrEqual:  0,
			LesserThan:         to.Unix(),
			GreaterThan:        0,
		},
	})

	var out []InvoiceCandidate
	for invoice, err := range list.All(ctx) {
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("list stripe invoices for subscription "+subscriptionID))
		}
		out = append(out, InvoiceCandidate{
			ID:            invoice.ID,
			Status:        InvoiceStatus(invoice.Status),
			BillingReason: string(invoice.BillingReason),
			PeriodStart:   invoice.PeriodStart,
			PeriodEnd:     invoice.PeriodEnd,
		})
	}
	return out, nil
}

func (r *stripeInvoiceReader) GetInvoice(ctx context.Context, invoiceID string) (Invoice, error) {
	invoice, err := r.client.V1Invoices.Retrieve(ctx, invoiceID, nil)
	if err != nil {
		return Invoice{}, fault.Wrap(err, fault.Internal("retrieve stripe invoice "+invoiceID)) //nolint:exhaustruct // zero-value return on the error path
	}

	lines, err := r.listLines(ctx, invoiceID)
	if err != nil {
		return Invoice{}, err //nolint:exhaustruct // zero-value return on the error path
	}
	return convertInvoice(invoice, lines), nil
}

// listLines fetches every line of one invoice, expanding each line's price so
// the price's lookup_key and id are available. The expand path goes through the
// per-invoice ListLines endpoint on purpose: the same expand through the
// invoice-list endpoint exceeds Stripe's 4-level expansion limit.
func (r *stripeInvoiceReader) listLines(ctx context.Context, invoiceID string) ([]InvoiceLine, error) {
	list := r.client.V1Invoices.ListLines(ctx, &stripe.InvoiceListLinesParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)}, //nolint:exhaustruct // only Limit is set; the rest default
		Invoice:    stripe.String(invoiceID),
		Expand:     []*string{stripe.String("data.pricing.price_details.price")},
	})

	var out []InvoiceLine
	for line, err := range list.All(ctx) {
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("list stripe invoice lines for "+invoiceID))
		}
		out = append(out, convertLine(line))
	}
	return out, nil
}

func convertLine(l *stripe.InvoiceLineItem) InvoiceLine {
	var priceID, lookupKey string
	var unitAmountDecimal float64
	if l.Pricing != nil && l.Pricing.PriceDetails != nil && l.Pricing.PriceDetails.Price != nil {
		p := l.Pricing.PriceDetails.Price
		priceID = p.ID
		lookupKey = p.LookupKey
		unitAmountDecimal = p.UnitAmountDecimal
	}

	var discount int64
	for _, d := range l.DiscountAmounts {
		discount += d.Amount
	}

	return InvoiceLine{
		ID:                  l.ID,
		AmountCents:         l.Amount,
		Quantity:            l.QuantityDecimal,
		PriceID:             priceID,
		PriceLookupKey:      lookupKey,
		UnitAmountDecimal:   unitAmountDecimal,
		DiscountAmountCents: discount,
	}
}

func convertInvoice(inv *stripe.Invoice, lines []InvoiceLine) Invoice {
	discounts := make([]DiscountAmount, 0, len(inv.TotalDiscountAmounts))
	for _, d := range inv.TotalDiscountAmounts {
		discounts = append(discounts, DiscountAmount{AmountCents: d.Amount})
	}

	credits := make([]PretaxCreditAmount, 0, len(inv.TotalPretaxCreditAmounts))
	for _, c := range inv.TotalPretaxCreditAmounts {
		credits = append(credits, PretaxCreditAmount{AmountCents: c.Amount, Type: PretaxCreditType(c.Type)})
	}

	taxes := make([]TaxAmount, 0, len(inv.TotalTaxes))
	for _, t := range inv.TotalTaxes {
		taxes = append(taxes, TaxAmount{AmountCents: t.Amount})
	}

	return Invoice{
		ID:                  inv.ID,
		Status:              InvoiceStatus(inv.Status),
		BillingReason:       string(inv.BillingReason),
		PeriodStart:         inv.PeriodStart,
		PeriodEnd:           inv.PeriodEnd,
		SubtotalCents:       inv.Subtotal,
		TotalCents:          inv.Total,
		AmountShippingCents: inv.AmountShipping,
		DiscountAmounts:     discounts,
		PretaxCreditAmounts: credits,
		Taxes:               taxes,
		Lines:               lines,
	}
}

// stripePriceReader implements PriceReader against the Stripe API.
type stripePriceReader struct {
	client *stripe.Client
}

var _ PriceReader = (*stripePriceReader)(nil)

// NewStripePriceReader builds a Stripe-backed PriceReader from a secret key.
func NewStripePriceReader(secretKey string) PriceReader {
	return &stripePriceReader{client: stripe.NewClient(secretKey)}
}

func (r *stripePriceReader) PriceByLookupKey(ctx context.Context, lookupKey string) (Price, error) {
	list := r.client.V1Prices.List(ctx, &stripe.PriceListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(1)}, //nolint:exhaustruct // only Limit and LookupKeys are set
		LookupKeys: []*string{stripe.String(lookupKey)},
	})

	for price, err := range list.All(ctx) {
		if err != nil {
			return Price{}, fault.Wrap(err, fault.Internal("list stripe prices for lookup_key "+lookupKey)) //nolint:exhaustruct // zero-value return on the error path
		}
		return Price{
			ID:                price.ID,
			LookupKey:         price.LookupKey,
			UnitAmountDecimal: price.UnitAmountDecimal,
		}, nil
	}
	return Price{}, ErrNotFound //nolint:exhaustruct // zero-value return; ErrNotFound is a defined "doesn't exist" outcome
}
