package billingreconcile

import (
	"context"
	"errors"
	"time"

	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// ErrNotFound is returned by a reader when the requested object genuinely does
// not exist upstream (no price for a lookup_key, no previous invoice), as
// distinct from a transient read failure. The engine treats the two very
// differently: a transient failure aborts the pass with an error, while a
// business-meaningful "not found" folds into the Result as a structural
// finding. See the failure-handling note on [Reconciler.ReconcileWorkspace].
var ErrNotFound = errors.New("billingreconcile: not found")

// Verdict is the reconcile pass's top-level classification for one workspace's
// billing period. Ordered by severity through [Verdict.severity]; when several
// checks fire, the most severe wins. The package only classifies, it does not
// act on the verdict.
type Verdict string

const (
	// VerdictClean: every check passed within tolerance.
	VerdictClean Verdict = "clean"
	// VerdictLateDataUnderbill: live ClickHouse usage exceeds what was invoiced
	// by more than the materiality bar on at least one meter, and nothing worse
	// was found. The expected, benign class (usage that settled after the
	// close's final push), so it carries a real dollar floor rather than paging
	// on any amount.
	VerdictLateDataUnderbill Verdict = "late_data_underbill"
	// VerdictOverbill: the invoice charged for more than live ClickHouse ever
	// recorded, on at least one meter beyond rounding noise, or applied less
	// credit than the customer was entitled to. Structurally near-impossible
	// under convergent last-aggregation pushes, so any occurrence gets no
	// threshold.
	VerdictOverbill Verdict = "overbill"
	// VerdictStructural: existence, price, credit over-application, or
	// total-decomposition failed -- a code bug or catalog drift, not a
	// usage-drift money decision. Takes precedence over every other verdict.
	VerdictStructural Verdict = "structural"
)

// severity ranks verdicts so findings fold into the worst one.
func (v Verdict) severity() int {
	switch v {
	case VerdictStructural:
		return 3
	case VerdictOverbill:
		return 2
	case VerdictLateDataUnderbill:
		return 1
	case VerdictClean:
		return 0
	default:
		return 3 // an unknown class never ranks below structural
	}
}

// Check names which of the reconcile checks produced a finding.
type Check string

const (
	CheckExistence Check = "existence"
	CheckQuantity  Check = "quantity"
	CheckPrice     Check = "price"
	CheckCredit    Check = "credit"
	CheckTotal     Check = "total"
)

// Finding is one failed check, flat by design: a caller assembles the monthly
// Slack summary and drift metrics straight off []Finding without unpacking a
// per-check struct tree.
type Finding struct {
	// Check names the check that produced the finding.
	Check Check
	// Class is the verdict this finding contributes to the result.
	Class Verdict
	// Meter is set on per-meter findings (quantity, price, duplicate line),
	// empty otherwise.
	Meter Meter
	// DriftCents is the priced drift in integer cents where one is meaningful:
	// positive means the customer was billed that much more than recorded,
	// negative that much less. Zero for purely structural findings with no
	// dollar amount. Every value is priced to integer cents with round() before
	// comparison, never compared as a raw float.
	DriftCents int64
	// Detail is a human-readable explanation for logs and the Slack summary.
	Detail string
}

// Result is the full outcome of reconciling one (workspace, period): the
// top-level Verdict plus every finding that produced it. Findings is empty
// exactly when the verdict is clean.
type Result struct {
	WorkspaceID string
	Period      billingperiod.Period
	// InvoiceID is the reconciled cycle invoice, set only when existence found
	// exactly one finalized invoice for the period.
	InvoiceID string
	Verdict   Verdict
	Findings  []Finding
}

// WorkspaceRef identifies one workspace to reconcile. The caller (the cron
// pass) enumerates these from our DB; both fields are required.
type WorkspaceRef struct {
	WorkspaceID          string
	StripeSubscriptionID string
}

// InvoiceStatus mirrors Stripe's invoice status field, narrowed to the values
// this package reasons about.
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid          InvoiceStatus = "void"
)

// Finalized reports whether an invoice in this status was actually finalized
// and billed: draft never went out, void was finalized then reversed
// (workspace deleted, invoice corrected) so it no longer represents a real
// charge. Both read as "not billed" for the existence check.
func (s InvoiceStatus) Finalized() bool {
	switch s {
	case InvoiceStatusOpen, InvoiceStatusPaid, InvoiceStatusUncollectible:
		return true
	case InvoiceStatusDraft, InvoiceStatusVoid:
		return false
	default:
		return false
	}
}

// InvoiceCandidate is the coarse (no lines) view of a windowed invoice: enough
// for the existence check to classify and the credit check to find the funder.
type InvoiceCandidate struct {
	ID            string
	Status        InvoiceStatus
	BillingReason string // Stripe billing_reason, e.g. "subscription_cycle" / "subscription_create"
	PeriodStart   int64  // unix seconds
	PeriodEnd     int64  // unix seconds
}

// PretaxCreditType is the kind of pretax credit Stripe applied, mirroring
// Stripe's invoice_total_pretax_credit_amount.type.
type PretaxCreditType string

const (
	// PretaxCreditDiscount is a coupon/discount routed through the pretax-credit
	// bucket rather than total_discount_amounts.
	PretaxCreditDiscount PretaxCreditType = "discount"
	// PretaxCreditBalanceTransaction is a Stripe billing credit grant consumed
	// against this invoice: where grantDeployCreditsForInvoice's promotional
	// usage-credit grants land (web/apps/dashboard/lib/stripe/deployCredits.ts).
	PretaxCreditBalanceTransaction PretaxCreditType = "credit_balance_transaction"
)

// DiscountAmount is one entry of an invoice's total_discount_amounts.
type DiscountAmount struct {
	AmountCents int64
}

// PretaxCreditAmount is one entry of an invoice's total_pretax_credit_amounts.
type PretaxCreditAmount struct {
	AmountCents int64
	Type        PretaxCreditType
}

// TaxAmount is one entry of an invoice's total_taxes.
type TaxAmount struct {
	AmountCents int64
}

// InvoiceLine is one line of a finalized invoice: a metered usage line, a
// plan-fee line, or some other manual/legacy line this package does not
// classify.
type InvoiceLine struct {
	ID string
	// AmountCents is the line's gross amount before its own discounts, in the
	// invoice currency's smallest unit.
	AmountCents int64
	// Quantity is the line's full-precision decimal quantity (quantity_decimal),
	// not the integer-truncated Quantity field.
	Quantity float64
	// PriceID is the id of the Stripe price the line billed through
	// (price_details.price). Compared against the price resolved by lookup_key
	// so a reprice that moved the lookup_key to a new price object, while the
	// subscription still bills the old one, is caught.
	PriceID string
	// PriceLookupKey is the stable identity of the price that priced this line
	// ("usage.cpu_seconds", "plan.pro", ...), empty when the line has no
	// associated price with a lookup_key (a manual invoice item, or a price
	// never given one).
	PriceLookupKey string
	// UnitAmountDecimal is the per-unit rate in cents the line billed at
	// (unit_amount_decimal), checked against the pinned catalog rate.
	UnitAmountDecimal float64
	// DiscountAmountCents sums this line's own discount_amounts (coupons
	// distributed to line level), subtracted from AmountCents to get what was
	// charged for the line net of coupons.
	DiscountAmountCents int64
}

// Invoice is the finalized-invoice detail this package needs: line-level
// quantity/amount/price identity and invoice-level total decomposition.
type Invoice struct {
	ID            string
	Status        InvoiceStatus
	BillingReason string
	PeriodStart   int64 // unix seconds
	PeriodEnd     int64 // unix seconds

	SubtotalCents       int64
	TotalCents          int64
	AmountShippingCents int64
	DiscountAmounts     []DiscountAmount
	PretaxCreditAmounts []PretaxCreditAmount
	Taxes               []TaxAmount

	Lines []InvoiceLine
}

// Price is the live Stripe price for one of our lookup keys.
type Price struct {
	ID                string
	LookupKey         string
	UnitAmountDecimal float64 // cents per unit
}

// InvoiceReader reads the Stripe invoices needed to reconcile one workspace's
// subscription for one billing period, coarse-then-detail: a single windowed
// list of candidates carrying no line detail, then a targeted full-detail fetch
// for the one invoice the existence check picks (and, when it exists, the
// previous cycle invoice the credit check needs). Never expands lines across
// the whole window. Implemented against the Stripe API in production (see
// stripe.go); faked in tests.
type InvoiceReader interface {
	// ListInvoices lists every invoice on the subscription (any status, any
	// billing_reason) created in [from, to) -- wide enough for the existence check
	// to tell missing from misaligned, and for the credit check to see the
	// funding proration. Coarse fields only, no lines.
	ListInvoices(ctx context.Context, subscriptionID string, from, to time.Time) ([]InvoiceCandidate, error)

	// GetInvoice fetches one invoice's full detail: lines (quantity, amount,
	// price id + lookup_key, per-line discounts) and the invoice-level total
	// decomposition. Called only for the invoices the engine already picked out
	// of the candidate window, never for every candidate.
	GetInvoice(ctx context.Context, invoiceID string) (Invoice, error)
}

// PriceReader fetches a live Stripe Price by its stable lookup_key
// ("usage.cpu_seconds", ...). Implemented against the Stripe API in production
// (see stripe.go); faked in tests.
type PriceReader interface {
	// PriceByLookupKey returns ErrNotFound when no price carries that lookup_key
	// at all -- a catalog drift (renamed, never created) the price check must
	// surface, not swallow.
	PriceByLookupKey(ctx context.Context, lookupKey string) (Price, error)
}

// UsageReader re-derives one workspace's billable Deploy usage for a [start,
// end) window (the reconciled invoice's actual period) from live ClickHouse, in
// the same units the hourly push bills from.
type UsageReader interface {
	WorkspaceUsage(ctx context.Context, workspaceID string, start, end time.Time) (billingmeter.MeterValues, error)
}

// Reconciler is the comparison engine. It holds only the read seams; construct
// one per pass with New and call ReconcileWorkspace per workspace.
type Reconciler struct {
	invoices InvoiceReader
	prices   PriceReader
	usage    UsageReader
}

// New builds a Reconciler over the given read seams.
func New(invoices InvoiceReader, prices PriceReader, usage UsageReader) *Reconciler {
	return &Reconciler{invoices: invoices, prices: prices, usage: usage}
}

// Lookup-key namespaces, mirroring tools/pricing.PlanLookupPrefix /
// UsageLookupPrefix. Pinned here rather than imported: tools/pricing is a
// separate Go module (kept dependency-free so its tests need no SDK), so this
// main-module package cannot import it.
const (
	planLookupPrefix  = "plan."
	usageLookupPrefix = "usage."
)

// Meter names one of the five Deploy usage meters by its catalog key
// (tools/pricing/catalog.go Meter.Key). The Stripe price lookup_key is
// "usage.<Meter>".
type Meter string

const (
	MeterCPUSeconds       Meter = "cpu_seconds"
	MeterMemoryGiBSeconds Meter = "memory_gib_seconds"
	MeterEgressGiB        Meter = "egress_public_gib"
	MeterDiskGiBSeconds   Meter = "disk_gib_seconds"
	MeterActiveKeys       Meter = "active_keys"
)

// Meters lists every Deploy meter in stable display order. The engine iterates
// this so a meter missing from either side is compared (as zero) rather than
// silently skipped.
func Meters() []Meter {
	return []Meter{
		MeterCPUSeconds,
		MeterMemoryGiBSeconds,
		MeterEgressGiB,
		MeterDiskGiBSeconds,
		MeterActiveKeys,
	}
}

// LookupKey is the meter's Stripe price lookup_key ("usage.cpu_seconds").
func (m Meter) LookupKey() string { return usageLookupPrefix + string(m) }

// rateCents is the pinned catalog rate for a meter, in cents per unit. The
// single source is deploybilling's mirror of tools/pricing/catalog.go, reused
// here so quantities and prices are checked against the exact numbers the
// hourly push bills with, with no third copy to drift.
func rateCents(m Meter) float64 {
	switch m {
	case MeterCPUSeconds:
		return deploybilling.CentsPerCPUSecond
	case MeterMemoryGiBSeconds:
		return deploybilling.CentsPerMemoryGiBSecond
	case MeterEgressGiB:
		return deploybilling.CentsPerEgressGiB
	case MeterDiskGiBSeconds:
		return deploybilling.CentsPerDiskGiBSecond
	case MeterActiveKeys:
		return deploybilling.CentsPerActiveKey
	}
	return 0
}

// meterQuantities flattens re-derived MeterValues into per-meter float
// quantities so the engine loops over Meters() instead of switching per field.
func meterQuantities(v billingmeter.MeterValues) map[Meter]float64 {
	return map[Meter]float64{
		MeterCPUSeconds:       v.CPUSeconds,
		MeterMemoryGiBSeconds: v.MemoryGiBSeconds,
		MeterEgressGiB:        v.EgressGiB,
		MeterDiskGiBSeconds:   v.DiskGiBSeconds,
		MeterActiveKeys:       float64(v.ActiveKeys),
	}
}
