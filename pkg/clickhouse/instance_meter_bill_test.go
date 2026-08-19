package clickhouse_test

import (
	"context"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Catalog rates in USD per meter unit, copied from the Terraform source of
// truth (infra terraform/stripe/modules/catalog/prices.tf, usage_rates). The
// trailing cents value there is per-unit; these are the dollar equivalents the
// hand-computed bills below multiply against. If Terraform changes a rate,
// this test should be updated in lockstep — it is the canary that the meter
// units the query produces map to the dollars we intend to charge.
const (
	rateCPUSecond       = 0.000006944 // per CPU-second
	rateMemoryGiBSecond = 0.000003472 // per GiB-second
	rateEgressGiB       = 0.05        // per GiB
	rateDiskGiBSecond   = 0.00000006  // per GiB-second
)

// The worker's unit conversions (svc/ctrl/worker/cron/deploybilling/billing.go).
// Replicated here so the test asserts the same path end to end: query natural
// units -> meter units -> dollars.
const (
	secondsPerHour = 3600.0
	bytesPerGiB    = float64(int64(1024 * 1024 * 1024))
)

// TestMeterUsageHandComputedBill seeds a workload with a computable bill: one
// container holding 1 vCPU, 2 GiB memory, and 10 GiB disk steady for exactly
// one hour, plus 240 MiB of egress. It runs the production billing query and
// checks the meter units and the resulting dollar bill against values worked
// out by hand below, so a query change that alters either is caught here.
func TestMeterUsageHandComputedBill(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	require.NoError(t, conn.Ping(ctx))

	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour).UnixMilli()

	ws := uid.New(uid.WorkspacePrefix)
	resource := uid.New("res")
	c := newContainer()

	// 240 intervals of 15s = exactly one hour (241 samples). Steady gauges; the
	// CPU and egress counters climb linearly so every interval contributes the
	// same delta.
	const (
		intervals      = 240
		cpuUsecPerStep = 15_000_000  // 1 vCPU for 15s = 15s of CPU time
		egressPerStep  = 1024 * 1024 // 1 MiB/step -> 240 MiB = 0.234375 GiB exact
		memBytes       = 2 * gib
		diskBytes      = 10 * gib
	)
	var samples []schema.InstanceCheckpoint
	for i := 0; i <= intervals; i++ {
		samples = append(samples, c.sample(ws, resource, base+int64(i)*sampleGap, sampleValues{
			cpuUsec:     int64(i) * cpuUsecPerStep,
			egressBytes: int64(i) * egressPerStep,
			memoryBytes: memBytes,
			diskBytes:   diskBytes,
		}))
	}
	insertCheckpoints(t, ctx, conn, samples)

	rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
		WorkspaceID: ws,
		Start:       base - sampleGap,
		End:         base + int64(2*time.Hour/time.Millisecond),
	})
	require.NoError(t, err)
	u := findUsage(t, rows, resource)

	// Natural units straight from the query.
	// CPU: 240 deltas * 15s = 3600 CPU-seconds.
	require.InDelta(t, 3600.0, u.CPUSeconds, 1e-6)
	// Memory: 2 GiB integrated over 240*15s = 3600s -> 2 GiB-hours.
	require.InDelta(t, 2.0, u.MemoryGiBHours, 1e-9)
	// Disk: 10 GiB over the same hour -> 10 GiB-hours.
	require.InDelta(t, 10.0, u.DiskGiBHours, 1e-9)
	// Egress: 240 deltas of 1 MiB = 240 MiB = 0.234375 GiB exact.
	require.Equal(t, int64(egressPerStep*intervals), u.EgressBytes)

	// Convert to meter units exactly as the worker does, then to dollars.
	cpuSeconds := u.CPUSeconds
	memoryGiBSeconds := u.MemoryGiBHours * secondsPerHour
	diskGiBSeconds := u.DiskGiBHours * secondsPerHour
	egressGiB := float64(u.EgressBytes) / bytesPerGiB

	bill := cpuSeconds*rateCPUSecond +
		memoryGiBSeconds*rateMemoryGiBSecond +
		diskGiBSeconds*rateDiskGiBSecond +
		egressGiB*rateEgressGiB

	// Hand-computed:
	//   CPU    3600 s          * 0.000006944 = 0.02499840
	//   Memory 7200 GiB-s      * 0.000003472 = 0.02499840
	//   Disk   36000 GiB-s     * 0.00000006  = 0.00216000
	//   Egress 0.234375 GiB    * 0.05        = 0.01171875
	//   total                                 = 0.06387555
	require.InDelta(t, 0.06387555, bill, 1e-9,
		"hand-computed bill drifted: cpu=%.6f mem=%.6f disk=%.6f egress=%.6f",
		cpuSeconds, memoryGiBSeconds, diskGiBSeconds, egressGiB)
}

// TestMeterUsageStraddlesWindow checks that only the in-window deltas are
// billed when a container's life straddles the billing window edges: samples
// before Start and after End exist, but the counter delta and time integral
// must cover only [Start, End]. This is the month-boundary case (close stamps
// period_end - 1s; the next period starts fresh).
func TestMeterUsageStraddlesWindow(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	require.NoError(t, conn.Ping(ctx))

	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour).UnixMilli()

	ws := uid.New(uid.WorkspacePrefix)
	resource := uid.New("res")
	c := newContainer()

	// 9 samples 15s apart, counter +1 CPU-second per step. We query the middle
	// window [sample2, sample6], so only the deltas fully inside it count.
	var samples []schema.InstanceCheckpoint
	for i := 0; i < 9; i++ {
		samples = append(samples, c.sample(ws, resource, base+int64(i)*sampleGap, sampleValues{
			cpuUsec:     int64(i) * 1_000_000,
			memoryBytes: gib,
			diskBytes:   gib,
		}))
	}
	insertCheckpoints(t, ctx, conn, samples)

	// Window [start, end) = [base+2*gap, base+6*gap). The WHERE filters to the
	// in-window rows {2,3,4,5} first; the delta is lead() over that filtered
	// set, so kept pairs are (2,3),(3,4),(4,5) = 3 CPU-seconds. Sample 5's
	// successor (sample 6) is outside the window, so 5 contributes no delta:
	// an interval is billed only when both its endpoints fall in-window. At a
	// month boundary this drops at most one sub-sample interval (~15s) of usage
	// at the edge, which under-counts rather than over-charges.
	rows, err := client.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
		WorkspaceID: ws,
		Start:       base + 2*sampleGap,
		End:         base + 6*sampleGap,
	})
	require.NoError(t, err)
	u := findUsage(t, rows, resource)
	require.InDelta(t, 3.0, u.CPUSeconds, 1e-9)
}
