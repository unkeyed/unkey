// Package stripeconnect verifies that a Stripe connected account id actually
// belongs to Unkey's platform before it is persisted as a billing target.
// Accepting a caller-supplied acct_... string without verification would let
// a workspace member bind an arbitrary merchant's account and redirect
// billing dispatch to it.
package stripeconnect

import (
	"context"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Verifier proves control of a connected account before linking.
type Verifier interface {
	// VerifyConnectedAccount returns nil only when the account exists and is
	// connected to this platform.
	VerifyConnectedAccount(ctx context.Context, accountID string) error
}

type stripeVerifier struct {
	client *stripe.Client
}

var _ Verifier = (*stripeVerifier)(nil)

// NewStripeVerifier verifies accounts against Stripe using the platform
// secret key: retrieving an account succeeds only when it is connected to
// the platform the key belongs to.
func NewStripeVerifier(secretKey string) Verifier {
	return &stripeVerifier{client: stripe.NewClient(secretKey)}
}

func (v *stripeVerifier) VerifyConnectedAccount(ctx context.Context, accountID string) error {
	_, err := v.client.V1Accounts.GetByID(ctx, accountID, &stripe.AccountRetrieveParams{}) //nolint:exhaustruct
	if err != nil {
		return fault.Wrap(err, fault.Internal("stripe connected account verification failed"))
	}
	return nil
}

type disabledVerifier struct{}

var _ Verifier = (*disabledVerifier)(nil)

// NewDisabledVerifier is used when no platform Stripe key is configured:
// linking always fails closed rather than storing an unverified account.
func NewDisabledVerifier() Verifier {
	return &disabledVerifier{}
}

func (v *disabledVerifier) VerifyConnectedAccount(ctx context.Context, accountID string) error {
	return fault.New("stripe connect is not configured on this deployment")
}
