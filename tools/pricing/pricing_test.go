package pricing

import "testing"

// These tests pin every amount in catalog.go, with no network or Stripe call.
// Any change to an amount that is not matched here fails CI, so a pricing edit
// has to be made in the same PR as its test.
//
// The `want` maps re-state every amount as a literal on purpose: comparing
// Desired() against itself would be a tautology. The hand-written duplication is
// what catches a shifted decimal.

func TestPlanRates(t *testing.T) {
	plans := indexPlans(t)

	want := map[string]int64{
		"starter":  500,
		"pro":      2_500,
		"business": 5_000,
	}

	if len(plans) != len(want) {
		t.Fatalf("plan count = %d, want %d", len(plans), len(want))
	}

	for key, cents := range want {
		p, ok := plans[key]
		if !ok {
			t.Errorf("missing plan %q", key)
			continue
		}
		if p.AmountCents != cents {
			t.Errorf("plan %q = %d cents, want %d", key, p.AmountCents, cents)
		}
		if got, want := p.LookupKey(), "plan."+key; got != want {
			t.Errorf("plan %q lookup_key = %q, want %q", key, got, want)
		}
	}
}

func TestMeterRates(t *testing.T) {
	meters := indexMeters(t)

	// cents is unit_amount_decimal; a shifted decimal here is the regression we
	// guard against.
	want := map[string]struct {
		display string
		event   string
		cents   float64
		agg     Aggregation
	}{
		"cpu_seconds":        {"CPU seconds", "cpu_seconds", 0.0006944, AggregationLast},
		"memory_gib_seconds": {"Memory GiB-seconds", "memory_gib_seconds", 0.0003472, AggregationLast},
		"egress_public_gib":  {"Egress GiB", "egress_public_gib", 5, AggregationLast},
		"disk_gib_seconds":   {"Disk GiB-seconds", "disk_gib_seconds", 0.000006, AggregationLast},
		"active_keys":        {"Active keys", "active_keys", 0.2, AggregationLast},
	}

	if len(meters) != len(want) {
		t.Fatalf("meter count = %d, want %d", len(meters), len(want))
	}

	for key, w := range want {
		m, ok := meters[key]
		if !ok {
			t.Errorf("missing meter %q", key)
			continue
		}
		if m.CentsPerUnit != w.cents {
			t.Errorf("meter %q centsPerUnit = %v, want %v", key, m.CentsPerUnit, w.cents)
		}
		if m.DisplayName != w.display {
			t.Errorf("meter %q display = %q, want %q", key, m.DisplayName, w.display)
		}
		if m.EventName != w.event {
			t.Errorf("meter %q event = %q, want %q", key, m.EventName, w.event)
		}
		if got, want := m.LookupKey(), "usage."+key; got != want {
			t.Errorf("meter %q lookup_key = %q, want %q", key, got, want)
		}
		if m.Aggregation != w.agg {
			t.Errorf("meter %q aggregation = %q, want %q", key, m.Aggregation, w.agg)
		}
	}
}

// TestMeterAggregationValid rejects an empty or unknown aggregation. Stripe only
// accepts last/sum/count, and a zero value would silently send "".
func TestMeterAggregationValid(t *testing.T) {
	valid := map[Aggregation]bool{
		AggregationLast:  true,
		AggregationSum:   true,
		AggregationCount: true,
	}
	for _, m := range Desired().Meters {
		if !valid[m.Aggregation] {
			t.Errorf("meter %q has invalid aggregation %q", m.Key, m.Aggregation)
		}
	}
}

func TestAPIProductRates(t *testing.T) {
	products := indexAPIProducts(t)

	want := map[string]struct {
		name  string
		cents int64
		quota int64
	}{
		"pro_250k":          {"API Pro 250k", 2500, 250_000},
		"pro_500k":          {"API Pro 500k", 5000, 500_000},
		"pro_1m":            {"API Pro 1M", 7500, 1_000_000},
		"pro_2m":            {"API Pro 2M", 10000, 2_000_000},
		"pro_10m":           {"API Pro 10M", 25000, 10_000_000},
		"pro_50m":           {"API Pro 50M", 50000, 50_000_000},
		"pro_100m":          {"API Pro 100M", 100000, 100_000_000},
		"enterprise":        {"Enterprise", 225000, 400_000_000},
		"dedicated_support": {"Dedicated Support Channel", 12500, 0},
		"sla_fee":           {"SLA fee", 800000, 0},
	}
	if len(products) != len(want) {
		t.Fatalf("api product count = %d, want %d", len(products), len(want))
	}
	for key, w := range want {
		p, ok := products[key]
		if !ok {
			t.Errorf("missing api product %q", key)
			continue
		}
		if p.AmountCents != w.cents {
			t.Errorf("api product %q amount = %d cents, want %d", key, p.AmountCents, w.cents)
		}
		if p.Name != w.name {
			t.Errorf("api product %q name = %q, want %q", key, p.Name, w.name)
		}
		if p.QuotaRequestsPerMonth != w.quota {
			t.Errorf("api product %q quota = %d, want %d", key, p.QuotaRequestsPerMonth, w.quota)
		}
		if got := p.HasQuota(); got != (w.quota > 0) {
			t.Errorf("api product %q HasQuota = %v, want %v", key, got, w.quota > 0)
		}
	}
}

func TestNoDuplicateKeys(t *testing.T) {
	c := Desired()
	seen := map[string]bool{}
	for _, p := range c.Plans {
		if seen[p.LookupKey()] {
			t.Errorf("duplicate lookup_key %q", p.LookupKey())
		}
		seen[p.LookupKey()] = true
	}
	for _, m := range c.Meters {
		if seen[m.LookupKey()] {
			t.Errorf("duplicate lookup_key %q", m.LookupKey())
		}
		seen[m.LookupKey()] = true
	}
	apiKeys := map[string]bool{}
	for _, a := range c.APIProducts {
		if apiKeys[a.Key] {
			t.Errorf("duplicate api product key %q", a.Key)
		}
		apiKeys[a.Key] = true
	}
}

func indexPlans(t *testing.T) map[string]Plan {
	t.Helper()
	m := map[string]Plan{}
	for _, p := range Desired().Plans {
		m[p.Key] = p
	}
	return m
}

func indexMeters(t *testing.T) map[string]Meter {
	t.Helper()
	m := map[string]Meter{}
	for _, x := range Desired().Meters {
		m[x.Key] = x
	}
	return m
}

func indexAPIProducts(t *testing.T) map[string]APIProduct {
	t.Helper()
	m := map[string]APIProduct{}
	for _, x := range Desired().APIProducts {
		m[x.Key] = x
	}
	return m
}
