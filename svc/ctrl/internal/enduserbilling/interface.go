// Package enduserbilling defines the push-provider seam for billing a
// customer's end-users. Unlike billingmeter (which reports one workspace's
// month-to-date totals to Unkey's own Stripe account), this seam carries
// per-identity usage records priced by the customer's rate card, destined
// for the customer's own billing provider.
//
// Provider selection is a config-driven branch at the period-close cron call
// site, mirroring how billingmeter picks Stripe-or-noop; a registry earns its
// keep only when a second push adapter ships. The export path is pull-based
// (served by the v2 billing API) and needs no adapter here.
package enduserbilling

import "context"

// MeterPusher reports one workspace's per-identity billable usage for a
// closed period to the customer's billing provider. Implementations must be
// idempotent under re-runs of the same closed period. Returns the number of
// usage records pushed.
type MeterPusher interface {
	Push(ctx context.Context, req PushRequest) (int, error)
}

// PushRequest is one workspace's per-identity usage for one closed billing
// period.
type PushRequest struct {
	WorkspaceID string
	// ConnectedAccountID is the customer's Stripe connected account
	// (acct_...) the push targets, decrypted from the workspace's billing
	// settings. Empty for providers that do not need it.
	ConnectedAccountID string
	// Year and Month identify the closed billing period the records cover.
	Year  int
	Month int
	// Records is the per-identity usage to bill, one entry per identity.
	Records []UsageRecord
}

// UsageRecord is one end-user identity's billable usage and pricing context
// for the period.
type UsageRecord struct {
	IdentityID string
	ExternalID string
	// ProviderCustomerID is the provider-side customer reference for this
	// end-user (e.g. the Stripe customer id on the customer's connected
	// account), from the identity's billing binding.
	ProviderCustomerID string
	// RateCardID is the rate card resolved for this identity and period
	// (KTD7 precedence, recorded per R18).
	RateCardID string
	// Quantities per metered dimension.
	Verifications    int64
	SpentCredits     int64
	RatelimitsPassed int64
	// Amounts per dimension in whole cents (rounded half-up), computed by
	// the shared rate-card resolver so the push and the invoice-draft API
	// price identically (R19). Currency is the resolved card's ISO 4217 code.
	VerificationsCents int64
	CreditsCents       int64
	RatelimitsCents    int64
	Currency           string
}

// Positive reports whether the record carries any usage worth pushing.
func (r UsageRecord) Positive() bool {
	return r.Verifications > 0 || r.SpentCredits > 0 || r.RatelimitsPassed > 0
}
