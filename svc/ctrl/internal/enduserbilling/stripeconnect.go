package enduserbilling

import (
	"context"
	"errors"
	"fmt"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
)

// InvoiceItemRequest is one priced line item to place on the customer's
// connected account.
type InvoiceItemRequest struct {
	ConnectedAccountID string
	// CustomerID is the Stripe customer on the connected account.
	CustomerID  string
	AmountCents int64
	// Currency is a lowercase ISO 4217 code.
	Currency    string
	Description string
	// IdempotencyKey dedups near-term retries of the same line item.
	IdempotencyKey string
}

// InvoiceItemCreator abstracts the single Stripe operation the pusher needs,
// so the adapter is unit-testable without live Stripe.
type InvoiceItemCreator interface {
	CreateInvoiceItem(ctx context.Context, req InvoiceItemRequest) error
}

type stripeInvoiceItems struct {
	client *stripe.Client
}

var _ InvoiceItemCreator = (*stripeInvoiceItems)(nil)

// NewStripeInvoiceItems creates invoice items via the platform key with the
// Stripe-Account header, so items land on the connected account which stays
// merchant-of-record (KTD3, direct-charge model).
func NewStripeInvoiceItems(secretKey string) InvoiceItemCreator {
	return &stripeInvoiceItems{client: stripe.NewClient(secretKey)}
}

func (s *stripeInvoiceItems) CreateInvoiceItem(ctx context.Context, req InvoiceItemRequest) error {
	params := &stripe.InvoiceItemCreateParams{} //nolint:exhaustruct
	params.SetStripeAccount(req.ConnectedAccountID)
	params.IdempotencyKey = stripe.String(req.IdempotencyKey)
	params.Customer = stripe.String(req.CustomerID)
	params.Amount = stripe.Int64(req.AmountCents)
	params.Currency = stripe.String(req.Currency)
	params.Description = stripe.String(req.Description)

	_, err := s.client.V1InvoiceItems.Create(ctx, params)
	if err != nil {
		return fault.Wrap(err, fault.Internal("stripe invoice item create failed"))
	}
	return nil
}

// stripeConnectPusher implements MeterPusher by writing one invoice item per
// (identity, dimension) with a non-zero amount onto the connected account.
//
// This is the plan's Q6 "post priced invoice items directly" mechanism: the
// amounts come from Unkey's rate-card resolver (exact math, R19), so no
// Stripe meter, Price, or aggregation config is required on the connected
// account — which also removes the meter-aggregation question (Q2) and the
// 35-day meter-event timestamp window from this path entirely.
//
// Idempotency is layered (KTD5): the resolver pins the period's rate card so
// amounts are stable across re-runs, and each item carries a deterministic
// per-(workspace, identity, period, dimension) idempotency key so Stripe
// dedups retries within its idempotency window.
type stripeConnectPusher struct {
	items InvoiceItemCreator
}

var _ MeterPusher = (*stripeConnectPusher)(nil)

// NewStripeConnectPusher wires the connect pusher over an item creator.
func NewStripeConnectPusher(items InvoiceItemCreator) MeterPusher {
	return &stripeConnectPusher{items: items}
}

func (p *stripeConnectPusher) Push(ctx context.Context, req PushRequest) (int, error) {
	if req.ConnectedAccountID == "" {
		return 0, fault.New("push request has no connected account id")
	}

	period := fmt.Sprintf("%04d-%02d", req.Year, req.Month)
	pushed := 0
	var errs []error

	for _, record := range req.Records {
		if !record.Positive() {
			continue
		}
		// Only records that will actually produce a priced line item need a
		// provider customer. A free-tier identity has usage (Positive) but zero
		// billable cents, so nothing is invoiced and a missing customer is not
		// an error — skip it silently rather than reporting a spurious gap.
		if record.VerificationsCents <= 0 && record.CreditsCents <= 0 && record.RatelimitsCents <= 0 {
			continue
		}
		// A missing provider customer is an actionable configuration gap: the
		// identity has billable amounts that cannot be invoiced until the
		// customer sets the billing binding.
		if record.ProviderCustomerID == "" {
			errs = append(errs, fault.New(
				"identity "+record.IdentityID+" has billable usage but no provider customer id",
			))
			continue
		}

		for _, line := range []struct {
			dimension string
			quantity  int64
			cents     int64
		}{
			{dimension: "verifications", quantity: record.Verifications, cents: record.VerificationsCents},
			{dimension: "credits", quantity: record.SpentCredits, cents: record.CreditsCents},
			{dimension: "ratelimits", quantity: record.RatelimitsPassed, cents: record.RatelimitsCents},
		} {
			if line.cents <= 0 {
				continue
			}
			err := p.items.CreateInvoiceItem(ctx, InvoiceItemRequest{
				ConnectedAccountID: req.ConnectedAccountID,
				CustomerID:         record.ProviderCustomerID,
				AmountCents:        line.cents,
				Currency:           record.Currency,
				Description: fmt.Sprintf("Unkey %s %s: %d (rate card %s)",
					line.dimension, period, line.quantity, record.RateCardID),
				IdempotencyKey: fmt.Sprintf("unkey-eub-%s-%s-%s-%s",
					req.WorkspaceID, record.IdentityID, period, line.dimension),
			})
			if err != nil {
				errs = append(errs, err)
				continue
			}
			pushed++
		}
	}

	return pushed, errors.Join(errs...)
}
