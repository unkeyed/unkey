// Package pricing declares the Unkey Stripe billing catalog as typed Go data.
// Stripe is reconciled to match it.
//
// This file holds the schema: the types, their methods, and the fixed Stripe
// contracts. The catalog itself (the amounts) lives in catalog.go.
//
// The package is dependency-free (standard library only) so pricing_test.go can
// pin every amount with no network call and no SDK. The reconcile/export logic
// that talks to Stripe lives in the internal/* packages.
package pricing

// Lookup-key namespaces. A managed price is identified by its lookup_key, not
// the Stripe-generated id, so the app and this tool reference stable strings.
const (
	PlanLookupPrefix  = "plan."  // e.g. plan.pro
	UsageLookupPrefix = "usage." // e.g. usage.cpu_seconds
)

// ManagedBy tags objects this tool owns via Stripe metadata, so reconcile can
// tell them apart from hand-made or legacy objects it must not touch.
//
// The key is snake_case, matching Stripe metadata-key convention and the other
// keys here (stripe_customer_id, pricing_key); the value is kebab-case. Both are
// written onto live Stripe objects and matched by reconcile, so they are stable
// identifiers: do not rewrite them, or already-tagged objects orphan.
const (
	ManagedByKey   = "managed_by"
	ManagedByValue = "unkey-pricing"
)

// PlanSignalKey is the price metadata key naming the Deploy tier
// ("starter"/"pro"/"business"). The dashboard's Stripe webhook reads it off a
// subscription's plan-fee price to set workspaces.deploy_plan (detectDeployPlan
// in web/apps/dashboard/lib/stripe/deployPlan.ts). It fails closed: a plan-fee
// price without this key reads as "no Deploy plan", so every plan price must
// carry it, including the new price minted on a reprice. Do not drop it.
const PlanSignalKey = "plan"

// Fixed meter event contract: the billing worker maps usage to a customer via
// stripe_customer_id and carries the number in "value". Same for every meter;
// only the aggregation varies.
const (
	MeterCustomerMappingKey = "stripe_customer_id"
	MeterValuePayloadKey    = "value"
)

// APIRetentionDays is the logs/audit-logs retention quota tagged on API tier
// products (all 90/90 today).
const APIRetentionDays = 90

// All amounts in this catalog are in cents, Stripe's native unit, so the
// reconciler does no dollar/cent conversion. Flat fees are whole cents (int64);
// sub-cent metered rates are fractional cents (float64).

// Aggregation is how Stripe rolls up events posted to a meter within a billing
// period. Maps to default_aggregation.formula; the set is fixed by Stripe.
type Aggregation string

const (
	// AggregationLast keeps the last value received in the period. Use it when
	// the worker posts a running period-to-date total each tick (CPU, memory,
	// egress, disk, active keys today).
	AggregationLast Aggregation = "last"
	// AggregationSum adds every value received in the period. Use it when the
	// worker posts per-event deltas instead of a running total.
	AggregationSum Aggregation = "sum"
	// AggregationCount counts the events received in the period, ignoring value.
	AggregationCount Aggregation = "count"
)

// Plan is a Deploy plan tier: a licensed, flat monthly subscription fee.
type Plan struct {
	Key         string // stable id; lookup_key is "plan.<Key>"
	Name        string // Stripe product name / invoice line label
	AmountCents int64  // monthly fee in cents
}

// LookupKey is the stable price identity (e.g. "plan.pro").
func (p Plan) LookupKey() string { return PlanLookupPrefix + p.Key }

// Meter is a usage meter plus its metered price. One Stripe product per meter,
// so each usage line on an invoice gets a distinct label.
type Meter struct {
	Key         string      // stable id; lookup_key is "usage.<Key>"
	DisplayName string      // Stripe meter + product name
	EventName   string      // the stable contract the billing worker POSTs against
	Aggregation Aggregation // how Stripe rolls up events in a period
	// CentsPerUnit is unit_amount_decimal: the price per unit in cents. Rates are
	// sub-cent, so this is fractional (e.g. 0.0006944 cents). Pinned in pricing_test.go.
	CentsPerUnit float64
}

// LookupKey is the stable price identity (e.g. "usage.cpu_seconds").
func (m Meter) LookupKey() string { return UsageLookupPrefix + m.Key }

// APIProduct is a legacy Unkey API product (licensed, flat monthly). Tier
// products carry request/retention quota metadata; add-ons do not.
type APIProduct struct {
	Key                   string
	Name                  string
	AmountCents           int64
	QuotaRequestsPerMonth int64 // 0 means none (add-on, no quota metadata)
}

// HasQuota reports whether this product is a tier (carries quota metadata).
func (a APIProduct) HasQuota() bool { return a.QuotaRequestsPerMonth > 0 }

// Catalog is the full, environment-independent desired state. (Webhook endpoints
// are environment-specific and live in webhooks.go.)
type Catalog struct {
	Plans       []Plan
	Meters      []Meter
	APIProducts []APIProduct
}
