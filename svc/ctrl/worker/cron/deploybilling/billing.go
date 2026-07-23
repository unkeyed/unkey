package deploybilling

import (
	"context"
	"fmt"
	"math"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/billingperiod"
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
// Stripe. tools/pricing is a separate Go module, so this main-module code
// cannot import Desired() to read them directly; the values are pinned here and
// must track catalog.go (both are covered by pricing_test.go and the TS mirror
// in deployPricing.ts). The spend-cap check prices usage locally with these so
// it never calls Stripe. The meters bill flat per unit (no tiers), so spend is
// a plain dot product of MeterValues and these rates. Exported so the billing
// reconcile engine (svc/ctrl/internal/billingreconcile) prices the exact same
// rates the hourly push bills with, with no second copy of the rates to drift.
const (
	CentsPerCPUSecond       = 0.0006944
	CentsPerMemoryGiBSecond = 0.0003472
	CentsPerEgressGiB       = 5.0
	CentsPerDiskGiBSecond   = 0.000006
	CentsPerActiveKey       = 0.2
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
		v.ActiveKeys = r.ActiveKeys
		values[r.WorkspaceID] = v
	}
}

// FleetMeterValues reads instance and active-keys usage for
// [p.Start(), endMillis) as two journaled steps and returns the aggregated
// per-workspace MeterValues. workspaceIDs scopes the ClickHouse scan; nil
// scans the whole fleet. Exported so the spend-cap check prices the exact
// same values the hourly push bills, scoped to its budgeted set instead of
// re-aggregating everyone's month at its tight cadence.
func FleetMeterValues(
	ctx restate.ObjectContext,
	usage UsageReader,
	p billingperiod.Period,
	endMillis int64,
	workspaceIDs []string,
) (map[string]billingmeter.MeterValues, error) {
	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID:  "",
			WorkspaceIDs: workspaceIDs,
			Start:        p.Start().UnixMilli(),
			End:          endMillis,
		})
	}, restate.WithName("get period usage"))
	if err != nil {
		return nil, fmt.Errorf("get period usage: %w", err)
	}

	keyRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ActiveKeysUsage, error) {
		return usage.GetActiveKeysUsage(rc, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID:  "",
			WorkspaceIDs: workspaceIDs,
			Year:         p.Year,
			Month:        p.Month,
		})
	}, restate.WithName("get active keys"))
	if err != nil {
		return nil, fmt.Errorf("get active keys: %w", err)
	}

	values := AggregateUsage(rows)
	MergeActiveKeys(values, keyRows)
	return values, nil
}

// MicroCentsPerCent is the fixed-point scale for priced usage: one cent is
// 1,000,000 micro-cents. Priced amounts are integers in micro-cents so every
// comparison, threshold, and proto field downstream is exact integer math;
// the only float involved is the dot product below, quantized once at this
// boundary. Micro-cent resolution is four orders of magnitude finer than the
// smallest catalog rate applied to one unit, so the quantization is far below
// anything billable.
const MicroCentsPerCent = 1_000_000

// PriceMicroCents returns the month-to-date Deploy spend for one workspace's
// meter values, in integer micro-cents. The usage quantities are inherently
// fractional (ClickHouse sums, unit conversions), so the dot product runs in
// float64 and is rounded to the nearest micro-cent exactly once, here. This is
// the gross usage the hourly push reports, priced with the catalog rates, and
// the same figure the spend-cap check compares against the budget.
func PriceMicroCents(v billingmeter.MeterValues) int64 {
	cents := v.CPUSeconds*CentsPerCPUSecond +
		v.MemoryGiBSeconds*CentsPerMemoryGiBSecond +
		v.EgressGiB*CentsPerEgressGiB +
		v.DiskGiBSeconds*CentsPerDiskGiBSecond +
		float64(v.ActiveKeys)*CentsPerActiveKey
	return int64(math.Round(cents * MicroCentsPerCent))
}

// FormatDollars renders micro-cents as a dollar string, dropping the cents
// when the amount is a whole dollar: $25, but $18.75 stays $18.75. Fractions
// below a cent are truncated for display. Mirrors the dashboard's
// formatDollars (web/apps/dashboard/lib/fmt.ts) so figures we send (budget
// alerts) read the same as the billing page.
func FormatDollars(microCents int64) string {
	const microCentsPerDollar = 100 * MicroCentsPerCent
	if microCents%microCentsPerDollar == 0 {
		return fmt.Sprintf("$%d", microCents/microCentsPerDollar)
	}
	cents := microCents / MicroCentsPerCent
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}
