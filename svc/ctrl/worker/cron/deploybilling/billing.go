package deploybilling

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// UsageReader returns billable Deploy usage for a time window, one row per
// resource. Implemented by *clickhouse.Client; faked in tests. Kept narrow on
// purpose so the handler depends only on the one query it needs.
type UsageReader interface {
	GetInstanceMeterUsage(ctx context.Context, req clickhouse.GetInstanceMeterUsageRequest) ([]clickhouse.InstanceMeterUsage, error)
}

const (
	// secondsPerHour converts the query's GiB-hour integrals to the
	// GiB-second unit the memory and disk meters bill in.
	secondsPerHour = 3600.0
	// bytesPerGiB converts egress bytes to binary GiB; the egress meter is
	// deploy.egress_public_gib (GiB, 2^30), not decimal GB.
	bytesPerGiB = 1024 * 1024 * 1024
)

// usageAccumulator sums per-resource meter rows for one workspace, in the
// query's natural units (converted to meter units in aggregateUsage).
type usageAccumulator struct {
	cpuSeconds     float64
	memoryGiBHours float64
	diskGiBHours   float64
	egressBytes    int64
}

// aggregateUsage sums the per-resource meter rows into per-workspace meter
// values, converting each meter from the query's natural unit into the unit
// its meter expects. Values stay full-precision (the meter events carry decimal
// strings), so there is no rounding here.
func aggregateUsage(rows []clickhouse.InstanceMeterUsage) map[string]billingmeter.MeterValues {
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
		}
	}
	return out
}
