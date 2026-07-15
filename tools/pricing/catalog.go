package pricing

// Desired is the catalog we want Stripe to reflect. Pure data: edit an amount
// here and update the matching pin in pricing_test.go in the same PR. Types and
// Stripe contracts live in pricing.go.
func Desired() Catalog {
	return Catalog{
		Plans: []Plan{
			{Key: "starter", Name: "Starter", AmountCents: 500},     // $5/mo
			{Key: "pro", Name: "Pro", AmountCents: 2_500},           // $25/mo
			{Key: "business", Name: "Business", AmountCents: 5_000}, // $50/mo
		},
		Meters: []Meter{
			{Key: "cpu_seconds", DisplayName: "CPU seconds", EventName: "cpu_seconds", Aggregation: AggregationLast, CentsPerUnit: 0.0006944},                      // $0.000006944 / vCPU-second
			{Key: "memory_gib_seconds", DisplayName: "Memory GiB-seconds", EventName: "memory_gib_seconds", Aggregation: AggregationLast, CentsPerUnit: 0.0003472}, // $0.000003472 / GiB-second
			{Key: "egress_public_gib", DisplayName: "Egress GiB", EventName: "egress_public_gib", Aggregation: AggregationLast, CentsPerUnit: 5},                   // $0.05 / GiB
			{Key: "disk_gib_seconds", DisplayName: "Disk GiB-seconds", EventName: "disk_gib_seconds", Aggregation: AggregationLast, CentsPerUnit: 0.000006},        // $0.00000006 / GiB-second
			{Key: "active_keys", DisplayName: "Active keys", EventName: "active_keys", Aggregation: AggregationLast, CentsPerUnit: 0.2},                            // $0.002 / active key
		},
		APIProducts: []APIProduct{
			{Key: "pro_250k", Name: "API Pro 250k", AmountCents: 2_500, QuotaRequestsPerMonth: 250_000},       // $25/mo
			{Key: "pro_500k", Name: "API Pro 500k", AmountCents: 5_000, QuotaRequestsPerMonth: 500_000},       // $50/mo
			{Key: "pro_1m", Name: "API Pro 1M", AmountCents: 7_500, QuotaRequestsPerMonth: 1_000_000},         // $75/mo
			{Key: "pro_2m", Name: "API Pro 2M", AmountCents: 10_000, QuotaRequestsPerMonth: 2_000_000},        // $100/mo
			{Key: "pro_10m", Name: "API Pro 10M", AmountCents: 25_000, QuotaRequestsPerMonth: 10_000_000},     // $250/mo
			{Key: "pro_50m", Name: "API Pro 50M", AmountCents: 50_000, QuotaRequestsPerMonth: 50_000_000},     // $500/mo
			{Key: "pro_100m", Name: "API Pro 100M", AmountCents: 100_000, QuotaRequestsPerMonth: 100_000_000}, // $1,000/mo
			{Key: "enterprise", Name: "Enterprise", AmountCents: 225_000, QuotaRequestsPerMonth: 400_000_000}, // $2,250/mo
			{Key: "dedicated_support", Name: "Dedicated Support Channel", AmountCents: 12_500},                // $125/mo
			{Key: "sla_fee", Name: "SLA fee", AmountCents: 800_000},                                           // $8,000/mo
		},
	}
}
