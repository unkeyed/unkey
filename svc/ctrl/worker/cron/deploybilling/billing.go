package deploybilling

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// UsageReader is the one ClickHouse query the handler needs.
type UsageReader interface {
	GetInstanceMeterUsage(ctx context.Context, req clickhouse.GetInstanceMeterUsageRequest) ([]clickhouse.InstanceMeterUsage, error)
}

const (
	secondsPerHour = 3600.0             // GiB-hours to GiB-seconds
	bytesPerGiB    = 1024 * 1024 * 1024 // binary GiB for egress meter
)

// usageAccumulator sums meter rows per workspace before unit conversion.
type usageAccumulator struct {
	cpuSeconds     float64
	memoryGiBHours float64
	diskGiBHours   float64
	egressBytes    int64
}

// aggregateUsage converts query units to Stripe meter units. No rounding here.
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
