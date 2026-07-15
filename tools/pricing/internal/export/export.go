// Package export renders the live Stripe catalog into the env block the
// dashboard consumes.
package export

import (
	"context"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v86"

	"github.com/unkeyed/unkey/tools/pricing"
	"github.com/unkeyed/unkey/tools/pricing/internal/stripeenv"
)

// meterEnvSuffix maps a meter key to the dashboard env var suffix
// (STRIPE_LOOKUP_DEPLOY_METER_<suffix>). Unknown meters fall back to the
// upper-cased key.
var meterEnvSuffix = map[string]string{
	"cpu_seconds":        "CPU",
	"memory_gib_seconds": "MEMORY",
	"egress_public_gib":  "EGRESS",
	"disk_gib_seconds":   "DISK",
	"active_keys":        "ACTIVE_KEYS",
}

// Render reads the live catalog for the client's environment and returns the env
// block as ordered "KEY=value" lines. The webhook signing secret is not included
// (Stripe never returns it on read); pass any secret created during an apply via
// extraSecrets to have it appended.
func Render(ctx context.Context, sc *stripeenv.Client, extraSecrets map[string]string) (string, error) {
	cat := pricing.Desired()
	var lines []string

	// Plan-fee price lookup_keys. The dashboard resolves these to the current
	// active price at runtime, so a reprice (new price, transferred lookup_key)
	// needs no re-export. reconcile guarantees the prices exist.
	for _, p := range cat.Plans {
		lines = append(lines, fmt.Sprintf("STRIPE_LOOKUP_DEPLOY_%s=%s", strings.ToUpper(p.Key), p.LookupKey()))
	}

	// Metered price lookup_keys, resolved by the dashboard the same way.
	for _, m := range cat.Meters {
		suffix, ok := meterEnvSuffix[m.Key]
		if !ok {
			suffix = strings.ToUpper(m.Key)
		}
		lines = append(lines, fmt.Sprintf("STRIPE_LOOKUP_DEPLOY_METER_%s=%s", suffix, m.LookupKey()))
	}

	// API products as two CSVs of product ids (the dashboard resolves each
	// product's default_price at runtime). pro_* -> PRO, enterprise -> ENTERPRISE;
	// add-ons like sla_fee are not exported.
	productIDs, err := apiProductIDs(ctx, sc, cat)
	if err != nil {
		return "", err
	}
	var pro, ent []string
	for _, a := range cat.APIProducts {
		switch id := productIDs[a.Key]; {
		case id == "":
			continue
		case strings.HasPrefix(a.Key, "pro_"):
			pro = append(pro, id)
		case strings.HasPrefix(a.Key, "enterprise"):
			ent = append(ent, id)
		}
	}
	if len(pro) > 0 {
		lines = append(lines, "STRIPE_PRODUCT_IDS_PRO="+strings.Join(pro, ","))
	}
	if len(ent) > 0 {
		lines = append(lines, "STRIPE_PRODUCT_IDS_ENTERPRISE="+strings.Join(ent, ","))
	}

	// Webhook secret, present only when the dashboard endpoint was created this run.
	if secret := extraSecrets["dashboard"]; secret != "" {
		lines = append(lines, "STRIPE_WEBHOOK_SECRET="+secret)
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func apiProductIDs(ctx context.Context, sc *stripeenv.Client, cat pricing.Catalog) (map[string]string, error) {
	byKey := map[string]string{}  // pricing_key -> product id
	byName := map[string]string{} // product name -> product id (adoption fallback)
	params := &stripe.ProductListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)}, // page size; .All paginates past it
		Active:     stripe.Bool(true),
	}
	for prod, err := range sc.V1Products.List(ctx, params).All(ctx) {
		if err != nil {
			return nil, err
		}
		if prod.Metadata[pricing.ManagedByKey] == pricing.ManagedByValue {
			if k := prod.Metadata["pricing_key"]; k != "" {
				byKey[k] = prod.ID
			}
		}
		if _, ok := byName[prod.Name]; !ok {
			byName[prod.Name] = prod.ID
		}
	}
	out := map[string]string{}
	for _, a := range cat.APIProducts {
		if id, ok := byKey[a.Key]; ok {
			out[a.Key] = id
		} else if id, ok := byName[a.Name]; ok {
			out[a.Key] = id
		}
	}
	return out, nil
}
