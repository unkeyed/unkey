package reconcile

// This file is the Stripe I/O layer: every read and write lives here, so
// reconcile.go reads as pure decision logic. Writes are additive only: create a
// price, transfer its lookup_key, archive the old one. Never delete.

import (
	"github.com/stripe/stripe-go/v86"

	"github.com/unkeyed/unkey/tools/pricing"
)

const (
	currencyUSD  = "usd"
	intervalMnth = "month"
	usageLicense = "licensed"
	usageMetered = "metered"
)

// ── Snapshot ─────────────────────────────────────────────────────────────────

// snapshot is a one-shot read of the live Stripe objects the reconciler cares
// about. Loading it once turns every per-catalog-entry lookup into a map hit,
// avoiding an N+1 of a List call inside each ensure*.
type snapshot struct {
	prices         []*stripe.Price
	pricesByLookup map[string]*stripe.Price

	products       []*stripe.Product
	productsByName map[string]*stripe.Product
	productsByKey  map[string]*stripe.Product // managed_by=unkey-pricing + pricing_key

	metersByEvent map[string]*stripe.BillingMeter
	webhooksByURL map[string]*stripe.WebhookEndpoint
}

// loadSnapshot reads active prices, active products, active meters, and webhook
// endpoints: four paginated lists total, regardless of catalog size.
func (r *reconciler) loadSnapshot() error {
	s := &snapshot{
		pricesByLookup: map[string]*stripe.Price{},
		productsByName: map[string]*stripe.Product{},
		productsByKey:  map[string]*stripe.Product{},
		metersByEvent:  map[string]*stripe.BillingMeter{},
		webhooksByURL:  map[string]*stripe.WebhookEndpoint{},
	}

	priceParams := &stripe.PriceListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)}, // page size; .All paginates past it
		Active:     stripe.Bool(true),
	}
	priceParams.AddExpand("data.product")
	for price, err := range r.sc.V1Prices.List(r.ctx, priceParams).All(r.ctx) {
		if err != nil {
			return err
		}
		s.prices = append(s.prices, price)
		if price.LookupKey != "" {
			s.pricesByLookup[price.LookupKey] = price
		}
	}

	prodParams := &stripe.ProductListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)}, // page size; .All paginates past it
		Active:     stripe.Bool(true),
	}
	prodParams.AddExpand("data.default_price")
	for prod, err := range r.sc.V1Products.List(r.ctx, prodParams).All(r.ctx) {
		if err != nil {
			return err
		}
		s.products = append(s.products, prod)
		if prod.Metadata[pricing.ManagedByKey] == pricing.ManagedByValue {
			if k := prod.Metadata["pricing_key"]; k != "" {
				s.productsByKey[k] = prod
			}
		}
		if _, ok := s.productsByName[prod.Name]; !ok {
			s.productsByName[prod.Name] = prod // first match wins
		}
	}

	meterParams := &stripe.BillingMeterListParams{Status: stripe.String("active")}
	for m, err := range r.sc.V1BillingMeters.List(r.ctx, meterParams).All(r.ctx) {
		if err != nil {
			return err
		}
		if _, ok := s.metersByEvent[m.EventName]; !ok {
			s.metersByEvent[m.EventName] = m
		}
	}

	for w, err := range r.sc.V1WebhookEndpoints.List(r.ctx, &stripe.WebhookEndpointListParams{}).All(r.ctx) {
		if err != nil {
			return err
		}
		if _, ok := s.webhooksByURL[w.URL]; !ok {
			s.webhooksByURL[w.URL] = w
		}
	}

	r.snap = s
	return nil
}

// ── Queries (in-memory, against the snapshot) ────────────────────────────────

func (r *reconciler) activePriceByLookupKey(lookupKey string) *stripe.Price {
	return r.snap.pricesByLookup[lookupKey]
}

func (r *reconciler) meterByEventName(event string) *stripe.BillingMeter {
	return r.snap.metersByEvent[event]
}

// apiProductByKeyOrName prefers the managed-metadata match (a product already
// adopted under this key), then falls back to an exact name match so a catalog
// built before this tool can be brought under management on first run.
//
// The name fallback never adopts a product already managed under a different
// key: matching on name alone would otherwise hijack another managed product
// that merely shares a catalog name. Untagged products are still adopted, which
// is the intended migration path.
func (r *reconciler) apiProductByKeyOrName(a pricing.APIProduct) *stripe.Product {
	if p, ok := r.snap.productsByKey[a.Key]; ok {
		return p
	}
	if p, ok := r.snap.productsByName[a.Name]; ok &&
		p.Metadata[pricing.ManagedByKey] != pricing.ManagedByValue {
		return p
	}
	return nil
}

func (r *reconciler) webhookByURL(url string) *stripe.WebhookEndpoint {
	return r.snap.webhooksByURL[url]
}

// ── Writes ───────────────────────────────────────────────────────────────────

func (r *reconciler) createProduct(name string, meta map[string]string) (*stripe.Product, error) {
	return r.sc.V1Products.Create(r.ctx, &stripe.ProductCreateParams{
		Name:     stripe.String(name),
		Metadata: meta,
	})
}

func (r *reconciler) createLicensedPrice(productID, lookupKey string, cents int64, nickname string, meta map[string]string) (*stripe.Price, error) {
	params := &stripe.PriceCreateParams{
		Product:    stripe.String(productID),
		Currency:   stripe.String(currencyUSD),
		UnitAmount: stripe.Int64(cents),
		Nickname:   stripe.String(nickname),
		Metadata:   meta,
		Recurring: &stripe.PriceCreateRecurringParams{
			Interval:  stripe.String(intervalMnth),
			UsageType: stripe.String(usageLicense),
		},
	}
	if lookupKey != "" {
		params.LookupKey = stripe.String(lookupKey)
		params.TransferLookupKey = stripe.Bool(true)
	}
	return r.sc.V1Prices.Create(r.ctx, params)
}

func (r *reconciler) createMeteredPrice(productID, meterID, lookupKey string, cents float64, nickname string, meta map[string]string) error {
	_, err := r.sc.V1Prices.Create(r.ctx, &stripe.PriceCreateParams{
		Product:           stripe.String(productID),
		Currency:          stripe.String(currencyUSD),
		BillingScheme:     stripe.String("per_unit"),
		UnitAmountDecimal: stripe.Float64(cents),
		LookupKey:         stripe.String(lookupKey),
		TransferLookupKey: stripe.Bool(true),
		Nickname:          stripe.String(nickname),
		Metadata:          meta,
		Recurring: &stripe.PriceCreateRecurringParams{
			Interval:  stripe.String(intervalMnth),
			UsageType: stripe.String(usageMetered),
			Meter:     stripe.String(meterID),
		},
	})
	return err
}

func (r *reconciler) createMeter(m pricing.Meter) (*stripe.BillingMeter, error) {
	return r.sc.V1BillingMeters.Create(r.ctx, &stripe.BillingMeterCreateParams{
		DisplayName: stripe.String(m.DisplayName),
		EventName:   stripe.String(m.EventName),
		DefaultAggregation: &stripe.BillingMeterCreateDefaultAggregationParams{
			Formula: stripe.String(string(m.Aggregation)),
		},
		CustomerMapping: &stripe.BillingMeterCreateCustomerMappingParams{
			Type:            stripe.String("by_id"),
			EventPayloadKey: stripe.String(pricing.MeterCustomerMappingKey),
		},
		ValueSettings: &stripe.BillingMeterCreateValueSettingsParams{
			EventPayloadKey: stripe.String(pricing.MeterValuePayloadKey),
		},
	})
}

func (r *reconciler) setDefaultPrice(productID, priceID string) error {
	_, err := r.sc.V1Products.Update(r.ctx, productID, &stripe.ProductUpdateParams{
		DefaultPrice: stripe.String(priceID),
	})
	return err
}

func (r *reconciler) archivePrice(priceID string) error {
	_, err := r.sc.V1Prices.Update(r.ctx, priceID, &stripe.PriceUpdateParams{Active: stripe.Bool(false)})
	return err
}
