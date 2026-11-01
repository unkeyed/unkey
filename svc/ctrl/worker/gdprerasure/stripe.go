package gdprerasure

import (
	"context"
	"errors"
	"net/http"

	stripe "github.com/stripe/stripe-go/v86"
)

// StripeCustomers deletes Stripe customers for a workspace erasure.
type StripeCustomers interface {
	// DeleteCustomer deletes the customer and reports whether it existed. A
	// customer that is already gone counts as deleted.
	DeleteCustomer(ctx context.Context, customerID string) (bool, error)
}

type stripeCustomers struct {
	client *stripe.Client
}

var _ StripeCustomers = (*stripeCustomers)(nil)

// NewStripe builds a Stripe-backed StripeCustomers from a secret key.
func NewStripe(secretKey string) StripeCustomers {
	return &stripeCustomers{client: stripe.NewClient(secretKey)}
}

func (s *stripeCustomers) DeleteCustomer(ctx context.Context, customerID string) (bool, error) {
	_, err := s.client.V1Customers.Delete(ctx, customerID, nil)
	if sErr, ok := errors.AsType[*stripe.Error](err); ok && (sErr.Code == stripe.ErrorCodeResourceMissing || sErr.HTTPStatusCode == http.StatusNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
