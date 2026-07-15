// Package reconcile diffs the desired pricing catalog against live Stripe and,
// in apply mode, makes Stripe match. It is additive and never deletes: a rate
// change creates a new immutable price, transfers the lookup_key onto it, and
// archives the old one. Managed objects absent from the catalog are reported as
// orphans, never touched.
package reconcile

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v86"

	"github.com/unkeyed/unkey/tools/pricing"
	"github.com/unkeyed/unkey/tools/pricing/internal/stripeenv"
)

// reconciler carries the per-run state.
type reconciler struct {
	ctx    context.Context
	sc     *stripeenv.Client
	apply  bool
	result *Result
	// snap is a one-shot read of live Stripe so per-object lookups are map hits
	// rather than a List call each.
	snap *snapshot
	// secrets collects webhook signing secrets created during this run (Stripe
	// only returns them at create time). Keyed by webhook Key.
	secrets map[string]string
	// onApply, if set, is called with each write as it lands so apply can stream
	// progress. nil in plan/verify mode.
	onApply func(Change)
}

// Run reconciles the catalog for the client's environment. With apply=false it
// only computes the diff (no writes). onApply, when non-nil, is called with each
// Stripe write as it completes so apply can stream progress; pass nil otherwise.
// It returns the result, any webhook secrets created during the run, and the
// first error encountered.
func Run(ctx context.Context, sc *stripeenv.Client, apply bool, onApply func(Change)) (*Result, map[string]string, error) {
	r := &reconciler{ctx: ctx, sc: sc, apply: apply, result: &Result{}, secrets: map[string]string{}, onApply: onApply}

	// One read of live Stripe up front; every ensure* below is then an in-memory
	// lookup instead of its own List call.
	if err := r.loadSnapshot(); err != nil {
		return r.result, r.secrets, fmt.Errorf("reading stripe state: %w", err)
	}

	cat := pricing.Desired()
	for _, p := range cat.Plans {
		n := len(r.result.Changes)
		if err := r.ensurePlan(p); err != nil {
			return r.result, r.secrets, fmt.Errorf("plan %s: %w", p.Key, err)
		}
		r.stream(n)
	}
	for _, m := range cat.Meters {
		n := len(r.result.Changes)
		if err := r.ensureMeter(m); err != nil {
			return r.result, r.secrets, fmt.Errorf("meter %s: %w", m.Key, err)
		}
		r.stream(n)
	}
	for _, a := range cat.APIProducts {
		n := len(r.result.Changes)
		if err := r.ensureAPIProduct(a); err != nil {
			return r.result, r.secrets, fmt.Errorf("api product %s: %w", a.Key, err)
		}
		r.stream(n)
	}
	for _, w := range pricing.Webhooks(sc.Env) {
		n := len(r.result.Changes)
		if err := r.ensureWebhook(w); err != nil {
			return r.result, r.secrets, fmt.Errorf("webhook %s: %w", w.Key, err)
		}
		r.stream(n)
	}
	r.detectOrphans(cat)
	return r.result, r.secrets, nil
}

// stream emits the changes added since index from. It runs after the ensure that
// produced them returns, so a streamed line means the write already landed; a
// write that fails is never streamed (the error names it instead). Noops and
// orphans are not writes and are never streamed.
func (r *reconciler) stream(from int) {
	if !r.apply || r.onApply == nil {
		return
	}
	for _, c := range r.result.Changes[from:] {
		if c.writesStripe() {
			r.onApply(c)
		}
	}
}

// ── Plans ────────────────────────────────────────────────────────────────────

func (r *reconciler) ensurePlan(p pricing.Plan) error {
	active := r.activePriceByLookupKey(p.LookupKey())
	// The plan-fee price carries plan=<tier> so the dashboard webhook can detect
	// the Deploy plan; see pricing.PlanSignalKey. This meta is reused on the
	// reprice path below, so the new price keeps the signal.
	meta := managedMeta("plan", p.Key, map[string]string{pricing.PlanSignalKey: p.Key})

	if active == nil {
		r.result.add(Change{ActionCreate, "plan", p.LookupKey(), usd(p.AmountCents)})
		if !r.apply {
			return nil
		}
		prod, err := r.createProduct(p.Name, managedMeta("plan", p.Key, nil))
		if err != nil {
			return err
		}
		_, err = r.createLicensedPrice(prod.ID, p.LookupKey(), p.AmountCents, p.Name+" plan fee", meta)
		return err
	}

	if active.UnitAmount == p.AmountCents {
		r.result.add(Change{ActionNoop, "plan", p.LookupKey(), usd(p.AmountCents)})
		return nil
	}

	r.result.add(Change{ActionReprice, "plan", p.LookupKey(),
		fmt.Sprintf("%s -> %s", usd(active.UnitAmount), usd(p.AmountCents))})
	if !r.apply {
		return nil
	}
	// The new price must live under the same product as the active one. Product
	// is an expandable field, so guard against a price returned without it
	// rather than panic on a nil dereference.
	if active.Product == nil {
		return fmt.Errorf("active price %s for %q has no product expanded; cannot reprice", active.ID, p.LookupKey())
	}
	if _, err := r.createLicensedPrice(active.Product.ID, p.LookupKey(), p.AmountCents, p.Name+" plan fee", meta); err != nil {
		return err
	}
	return r.archivePrice(active.ID)
}

// ── Meters + metered prices ──────────────────────────────────────────────────

// ensureMeter reconciles a meter together with its metered price. They are two
// Stripe objects but one thing to an operator (a usage rate), so this reports a
// single "usage" change, not one per object.
func (r *reconciler) ensureMeter(m pricing.Meter) error {
	meter := r.meterByEventName(m.EventName)
	active := r.activePriceByLookupKey(m.LookupKey())
	want := m.CentsPerUnit

	switch {
	// Something is missing: create whatever isn't there (meter, price, or both).
	case meter == nil || active == nil:
		r.result.add(Change{ActionCreate, "usage", m.LookupKey(), formatCents(want) + "¢/unit"})
		if !r.apply {
			return nil
		}
		if meter == nil {
			created, err := r.createMeter(m)
			if err != nil {
				return err
			}
			meter = created
		}
		if active == nil {
			prod, err := r.createProduct(m.DisplayName, managedMeta("meter", m.Key, nil))
			if err != nil {
				return err
			}
			return r.createMeteredPrice(prod.ID, meter.ID, m.LookupKey(), want, m.DisplayName, managedMeta("meter", m.Key, nil))
		}
		return nil

	// Rate changed: publish a new price, transfer the lookup_key, archive the old.
	case formatCents(active.UnitAmountDecimal) != formatCents(want):
		r.result.add(Change{ActionReprice, "usage", m.LookupKey(),
			fmt.Sprintf("%s -> %s ¢/unit", formatCents(active.UnitAmountDecimal), formatCents(want))})
		if !r.apply {
			return nil
		}
		meterID := meter.ID
		if active.Recurring != nil && active.Recurring.Meter != "" {
			meterID = active.Recurring.Meter
		}
		if err := r.createMeteredPrice(active.Product.ID, meterID, m.LookupKey(), want, m.DisplayName, managedMeta("meter", m.Key, nil)); err != nil {
			return err
		}
		return r.archivePrice(active.ID)

	default:
		r.result.add(Change{ActionNoop, "usage", m.LookupKey(), formatCents(want) + "¢/unit"})
		return nil
	}
}

// ── API products ─────────────────────────────────────────────────────────────

func (r *reconciler) ensureAPIProduct(a pricing.APIProduct) error {
	prod := r.apiProductByKeyOrName(a)
	meta := apiProductMeta(a)

	if prod == nil {
		r.result.add(Change{ActionCreate, "api_product", a.Key, usd(a.AmountCents)})
		if !r.apply {
			return nil
		}
		created, err := r.createProduct(a.Name, meta)
		if err != nil {
			return err
		}
		prod = created
		price, err := r.createLicensedPrice(prod.ID, "", a.AmountCents, a.Name, managedMeta("api", a.Key, nil))
		if err != nil {
			return err
		}
		return r.setDefaultPrice(prod.ID, price.ID)
	}

	var current *stripe.Price
	if prod.DefaultPrice != nil {
		current = prod.DefaultPrice
	}
	if current != nil && current.UnitAmount == a.AmountCents {
		r.result.add(Change{ActionNoop, "api_product", a.Key, usd(a.AmountCents)})
		return nil
	}

	from := "∅"
	if current != nil {
		from = usd(current.UnitAmount)
	}
	r.result.add(Change{ActionReprice, "api_product", a.Key, fmt.Sprintf("%s -> %s", from, usd(a.AmountCents))})
	if !r.apply {
		return nil
	}
	price, err := r.createLicensedPrice(prod.ID, "", a.AmountCents, a.Name, managedMeta("api", a.Key, nil))
	if err != nil {
		return err
	}
	if err := r.setDefaultPrice(prod.ID, price.ID); err != nil {
		return err
	}
	if _, err := r.sc.V1Products.Update(r.ctx, prod.ID, &stripe.ProductUpdateParams{Metadata: meta}); err != nil {
		return err
	}
	if current != nil {
		return r.archivePrice(current.ID)
	}
	return nil
}

// ── Webhooks ─────────────────────────────────────────────────────────────────

func (r *reconciler) ensureWebhook(w pricing.Webhook) error {
	existing := r.webhookByURL(w.URL)
	if existing == nil {
		r.result.add(Change{ActionCreate, "webhook", w.Key, w.URL})
		if !r.apply {
			return nil
		}
		created, err := r.sc.V1WebhookEndpoints.Create(r.ctx, &stripe.WebhookEndpointCreateParams{
			URL:           stripe.String(w.URL),
			EnabledEvents: stripe.StringSlice(w.EnabledEvents),
			Description:   stripe.String(w.Description),
		})
		if err != nil {
			return err
		}
		if created.Secret != "" {
			r.secrets[w.Key] = created.Secret // only available at create time
		}
		return nil
	}

	if sameStringSet(existing.EnabledEvents, w.EnabledEvents) {
		r.result.add(Change{ActionNoop, "webhook", w.Key, w.URL})
		return nil
	}
	r.result.add(Change{ActionUpdate, "webhook", w.Key, "enabled_events"})
	if !r.apply {
		return nil
	}
	_, err := r.sc.V1WebhookEndpoints.Update(r.ctx, existing.ID, &stripe.WebhookEndpointUpdateParams{
		EnabledEvents: stripe.StringSlice(w.EnabledEvents),
		Description:   stripe.String(w.Description),
	})
	return err
}

// ── Orphan detection ─────────────────────────────────────────────────────────

// detectOrphans flags managed objects with no matching catalog entry: an active
// price whose lookup_key is in our namespace but no longer declared, or a
// product tagged managed_by=unkey-pricing whose pricing_key was removed from
// catalog.go. Reported only, never touched: `plan` shows them, `verify` fails on
// them, an operator decides.
func (r *reconciler) detectOrphans(cat pricing.Catalog) {
	wantLookup := map[string]bool{}
	for _, p := range cat.Plans {
		wantLookup[p.LookupKey()] = true
	}
	for _, m := range cat.Meters {
		wantLookup[m.LookupKey()] = true
	}

	for _, price := range r.snap.prices {
		lk := price.LookupKey
		if lk == "" {
			continue
		}
		if !strings.HasPrefix(lk, pricing.PlanLookupPrefix) && !strings.HasPrefix(lk, pricing.UsageLookupPrefix) {
			continue
		}
		if !wantLookup[lk] {
			r.result.add(Change{ActionOrphan, "price", lk, "in Stripe, not in catalog"})
		}
	}

	wantKey := map[string]bool{}
	for _, p := range cat.Plans {
		wantKey[p.Key] = true
	}
	for _, m := range cat.Meters {
		wantKey[m.Key] = true
	}
	for _, a := range cat.APIProducts {
		wantKey[a.Key] = true
	}

	for _, prod := range r.snap.products {
		if prod.Metadata[pricing.ManagedByKey] != pricing.ManagedByValue {
			continue // hand-made or legacy product, not ours
		}
		key := prod.Metadata["pricing_key"]
		if key == "" || wantKey[key] {
			continue
		}
		r.result.add(Change{ActionOrphan, "product", key, "managed in Stripe, not in catalog"})
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func managedMeta(kind, key string, extra map[string]string) map[string]string {
	m := map[string]string{
		pricing.ManagedByKey: pricing.ManagedByValue,
		"pricing_kind":       kind,
		"pricing_key":        key,
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func apiProductMeta(a pricing.APIProduct) map[string]string {
	m := managedMeta("api", a.Key, nil)
	if a.HasQuota() {
		m["quota_requests_per_month"] = strconv.FormatInt(a.QuotaRequestsPerMonth, 10)
		m["quota_logs_retention_days"] = strconv.Itoa(pricing.APIRetentionDays)
		m["quota_audit_logs_retention_days"] = strconv.Itoa(pricing.APIRetentionDays)
	}
	return m
}

func formatCents(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// usd renders integer cents as a dollar string for the diff ("$25", "$29.99").
// Display only; the integer cents are what reach Stripe.
func usd(cents int64) string {
	if cents%100 == 0 {
		return fmt.Sprintf("$%d", cents/100)
	}
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, x := range a {
		seen[x]++
	}
	for _, x := range b {
		seen[x]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}
