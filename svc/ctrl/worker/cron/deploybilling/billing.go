package deploybilling

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// UsageReader returns billable Deploy usage for a time window, one row per
// resource. Implemented by *clickhouse.Client; faked in tests. Kept narrow on
// purpose so the handler depends only on the one query it needs.
type UsageReader interface {
	GetInstanceMeterUsage(ctx context.Context, req clickhouse.GetInstanceMeterUsageRequest) ([]clickhouse.InstanceMeterUsage, error)
	GetActiveKeysUsage(ctx context.Context, req clickhouse.GetActiveKeysUsageRequest) ([]clickhouse.ActiveKeysUsage, error)
}

const (
	// secondsPerHour converts the query's GiB-hour integrals to the
	// GiB-second unit the memory and disk meters bill in.
	secondsPerHour = 3600.0
	// bytesPerGiB converts egress bytes to binary GiB; the egress meter is
	// deploy.egress_public_gib (GiB, 2^30), not decimal GB.
	bytesPerGiB = 1024 * 1024 * 1024
)

// Per-unit Deploy meter rates in cents, mirroring the canonical catalog in
// tools/pricing/catalog.go (CentsPerUnit), which the reconciler writes to
// Stripe. The spend-cap check prices usage locally with these so it never calls
// Stripe; any drift from the catalog misprices the cap, so these values must
// track catalog.go. The meters bill flat per unit (no tiers), so spend is a
// plain dot product of MeterValues and these rates.
const (
	centsPerCPUSecond       = 0.0006944
	centsPerMemoryGiBSecond = 0.0003472
	centsPerEgressGiB       = 5.0
	centsPerDiskGiBSecond   = 0.000006
	centsPerActiveKey       = 0.2
)

// usageAccumulator sums per-resource meter rows for one workspace, in the
// query's natural units (converted to meter units in AggregateUsage).
type usageAccumulator struct {
	cpuSeconds     float64
	memoryGiBHours float64
	diskGiBHours   float64
	egressBytes    int64
}

// AggregateUsage sums the per-resource meter rows into per-workspace meter
// values, converting each meter from the query's natural unit into the unit
// its meter expects. Values stay full-precision (the meter events carry decimal
// strings), so there is no rounding here. Exported so the spend-cap check
// prices the exact same MeterValues the hourly push reports, with no second
// copy of the unit conversions to drift.
func AggregateUsage(rows []clickhouse.InstanceMeterUsage) map[string]billingmeter.MeterValues {
	sums := make(map[string]*usageAccumulator)
	for _, r := range rows {
		a := sums[r.WorkspaceID]
		if a == nil {
			a = &usageAccumulator{} //nolint:exhaustruct // zero-value accumulator, summed into below
			sums[r.WorkspaceID] = a
		}
		a.cpuSeconds += r.CPUSeconds
		a.memoryGiBHours += r.MemoryGiBHours
		a.diskGiBHours += r.DiskGiBHours
		a.egressBytes += r.EgressBytes
	}

	out := make(map[string]billingmeter.MeterValues, len(sums))
	for id, a := range sums {
		out[id] = billingmeter.MeterValues{
			CPUSeconds:       a.cpuSeconds,
			MemoryGiBSeconds: a.memoryGiBHours * secondsPerHour,
			EgressGiB:        float64(a.egressBytes) / bytesPerGiB,
			DiskGiBSeconds:   a.diskGiBHours * secondsPerHour,
			ActiveKeys:       0, // merged in from the active-keys query below
		}
	}
	return out
}

// MergeActiveKeys folds the per-workspace active-key counts into the meter
// values, adding entries for workspaces that have key activity but no
// instance usage (possible: a deployment can be scaled to zero while its
// keys keep verifying through the gateway).
func MergeActiveKeys(
	values map[string]billingmeter.MeterValues,
	rows []clickhouse.ActiveKeysUsage,
) {
	for _, r := range rows {
		v := values[r.WorkspaceID] // zero value when instance usage is absent
		v.ActiveKeys = float64(r.ActiveKeys)
		values[r.WorkspaceID] = v
	}
}

// PriceCents returns the month-to-date Deploy spend for one workspace's meter
// values, in fractional cents. No rounding: callers compare it against the
// integer budget directly, so a partial cent still counts. This is the gross
// usage the hourly push reports, priced with the catalog rates; the spend-cap
// check subtracts the included credit from it to get the budgeted overage.
func PriceCents(v billingmeter.MeterValues) float64 {
	return v.CPUSeconds*centsPerCPUSecond +
		v.MemoryGiBSeconds*centsPerMemoryGiBSecond +
		v.EgressGiB*centsPerEgressGiB +
		v.DiskGiBSeconds*centsPerDiskGiBSecond +
		v.ActiveKeys*centsPerActiveKey
}

// FormatDollars renders cents as a dollar string, dropping the cents when the
// amount is whole: $25, but $18.75 stays $18.75. Mirrors the dashboard's
// formatDollars (web/apps/dashboard/lib/fmt.ts) so figures we send (budget
// alerts) read the same as the billing page.
func FormatDollars(cents float64) string {
	if int64(cents)%100 == 0 && cents == float64(int64(cents)) {
		return fmt.Sprintf("$%d", int64(cents)/100)
	}
	return fmt.Sprintf("$%.2f", cents/100)
}
