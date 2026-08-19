package deploybilling

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

func TestInstanceMeterUsageShardsWorkspaceIDs(t *testing.T) {
	workspaceIDs := make([]string, 20)
	for i := range workspaceIDs {
		workspaceIDs[i] = fmt.Sprintf("ws_%02d", i)
	}

	requests := instanceMeterUsageShards(clickhouse.GetInstanceMeterUsageRequest{
		WorkspaceIDs: workspaceIDs,
		Start:        1,
		End:          2,
	})
	require.Len(t, requests, maxInstanceUsageShards)

	seen := make(map[string]int, len(workspaceIDs))
	for _, req := range requests {
		require.Equal(t, int64(1), req.Start)
		require.Equal(t, int64(2), req.End)
		for _, workspaceID := range req.WorkspaceIDs {
			seen[workspaceID]++
		}
	}
	for _, workspaceID := range workspaceIDs {
		require.Equal(t, 1, seen[workspaceID], "workspace must belong to exactly one shard")
	}
}

func TestAggregateUsage(t *testing.T) {
	const gib = 1 << 30

	t.Run("sums resources per workspace and converts to meter units", func(t *testing.T) {
		rows := []clickhouse.InstanceMeterUsage{
			// Two resources for ws_a, one for ws_b.
			{WorkspaceID: "ws_a", ResourceID: "r1", CPUSeconds: 10.5, MemoryGiBHours: 2.0, DiskGiBHours: 1.0, EgressBytes: gib},
			{WorkspaceID: "ws_a", ResourceID: "r2", CPUSeconds: 1.5, MemoryGiBHours: 0.5, DiskGiBHours: 0.0, EgressBytes: gib},
			{WorkspaceID: "ws_b", ResourceID: "r3", CPUSeconds: 100.0, MemoryGiBHours: 1.0, DiskGiBHours: 0.0, EgressBytes: 0},
		}

		out := AggregateUsage(rows)
		require.Len(t, out, 2)

		a := out["ws_a"]
		require.InDelta(t, 12.0, a.CPUSeconds, 1e-9)           // 10.5 + 1.5
		require.InDelta(t, 2.5*3600, a.MemoryGiBSeconds, 1e-6) // (2.0+0.5) GiB-h -> GiB-s
		require.InDelta(t, 1.0*3600, a.DiskGiBSeconds, 1e-6)   // 1.0 GiB-h -> GiB-s
		require.InDelta(t, 2.0, a.EgressGiB, 1e-9)             // 2 GiB of bytes -> 2 GiB

		b := out["ws_b"]
		require.InDelta(t, 100.0, b.CPUSeconds, 1e-9)
		require.InDelta(t, 1.0*3600, b.MemoryGiBSeconds, 1e-6)
		require.Zero(t, b.DiskGiBSeconds)
		require.Zero(t, b.EgressGiB)
	})

	t.Run("empty input yields empty map", func(t *testing.T) {
		require.Empty(t, AggregateUsage(nil))
	})
}

func TestMergeActiveKeys(t *testing.T) {
	values := map[string]billingmeter.MeterValues{
		"ws_with_usage": {CPUSeconds: 10, MemoryGiBSeconds: 0, EgressGiB: 0, DiskGiBSeconds: 0, ActiveKeys: 0},
	}
	MergeActiveKeys(values, []clickhouse.ActiveKeysUsage{
		{WorkspaceID: "ws_with_usage", ActiveKeys: 5},
		// Key activity without instance usage: deployment scaled to zero
		// while its keys keep verifying through the gateway.
		{WorkspaceID: "ws_keys_only", ActiveKeys: 2},
	})

	require.Equal(t, int64(5), values["ws_with_usage"].ActiveKeys)
	require.Equal(t, 10.0, values["ws_with_usage"].CPUSeconds, "existing meters must survive the merge")
	require.Equal(t, int64(2), values["ws_keys_only"].ActiveKeys)
	require.True(t, values["ws_keys_only"].Positive())
}

func TestPriceMicroCents(t *testing.T) {
	t.Run("zero usage costs nothing", func(t *testing.T) {
		require.Zero(t, PriceMicroCents(billingmeter.MeterValues{}))
	})

	t.Run("each meter priced at its catalog rate", func(t *testing.T) {
		// One unit of each meter in isolation must equal that meter's
		// CentsPerUnit from tools/pricing/catalog.go, in micro-cents. Exact
		// equality: the quantization happens inside PriceMicroCents, so the
		// contract is integers out.
		require.Equal(t, int64(694), PriceMicroCents(billingmeter.MeterValues{CPUSeconds: 1}))
		require.Equal(t, int64(347), PriceMicroCents(billingmeter.MeterValues{MemoryGiBSeconds: 1}))
		require.Equal(t, int64(5_000_000), PriceMicroCents(billingmeter.MeterValues{EgressGiB: 1}))
		require.Equal(t, int64(6), PriceMicroCents(billingmeter.MeterValues{DiskGiBSeconds: 1}))
		require.Equal(t, int64(200_000), PriceMicroCents(billingmeter.MeterValues{ActiveKeys: 1}))
	})

	t.Run("meters sum", func(t *testing.T) {
		// $0.50 plan-month of egress (10 GiB) plus 100 active keys ($0.20):
		// 50 cents + 20 cents = 70,000,000 micro-cents.
		got := PriceMicroCents(billingmeter.MeterValues{EgressGiB: 10, ActiveKeys: 100})
		require.Equal(t, int64(70*MicroCentsPerCent), got)
	})

	t.Run("rounds to the nearest micro-cent", func(t *testing.T) {
		// 1 CPU-second prices to 694.4 micro-cents; rounding, not truncation,
		// so 2 CPU-seconds is 1,389 (1388.8 rounded up), not 1,388.
		require.Equal(t, int64(1_389), PriceMicroCents(billingmeter.MeterValues{CPUSeconds: 2}))
	})
}

func TestFormatDollars(t *testing.T) {
	require.Equal(t, "$25", FormatDollars(2_500*MicroCentsPerCent))
	require.Equal(t, "$18.75", FormatDollars(1_875*MicroCentsPerCent))
	require.Equal(t, "$0.01", FormatDollars(1*MicroCentsPerCent))
	// Sub-cent fractions are truncated for display.
	require.Equal(t, "$18.75", FormatDollars(1_875*MicroCentsPerCent+499_999))
	require.Equal(t, "$0", FormatDollars(0))
}

func TestUsageIngestionDelay(t *testing.T) {
	p, err := billingperiod.Parse("2026-06")
	require.NoError(t, err)

	t.Run("waits for the remainder of the lateness window after period end", func(t *testing.T) {
		now := p.End().Add(2 * time.Hour)
		require.Equal(t, 22*time.Hour, usageIngestionDelay(p, now))
	})

	t.Run("small future clock skew still waits through period end", func(t *testing.T) {
		now := p.End().Add(-time.Second)
		require.Equal(t, usageIngestLateness+time.Second, usageIngestionDelay(p, now))
	})

	t.Run("test clock period far ahead of wall time skips the wall-clock wait", func(t *testing.T) {
		now := p.End().Add(-7 * 24 * time.Hour)
		require.Zero(t, usageIngestionDelay(p, now))
	})
}
