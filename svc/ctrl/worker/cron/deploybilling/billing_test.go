package deploybilling

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

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

	require.Equal(t, 5.0, values["ws_with_usage"].ActiveKeys)
	require.Equal(t, 10.0, values["ws_with_usage"].CPUSeconds, "existing meters must survive the merge")
	require.Equal(t, 2.0, values["ws_keys_only"].ActiveKeys)
	require.True(t, values["ws_keys_only"].Positive())
}

func TestPriceCents(t *testing.T) {
	t.Run("zero usage costs nothing", func(t *testing.T) {
		require.Zero(t, PriceCents(billingmeter.MeterValues{}))
	})

	t.Run("each meter priced at its catalog rate", func(t *testing.T) {
		// One unit of each meter in isolation must equal that meter's
		// CentsPerUnit from tools/pricing/catalog.go.
		require.InDelta(t, 0.0006944, PriceCents(billingmeter.MeterValues{CPUSeconds: 1}), 1e-12)
		require.InDelta(t, 0.0003472, PriceCents(billingmeter.MeterValues{MemoryGiBSeconds: 1}), 1e-12)
		require.InDelta(t, 5.0, PriceCents(billingmeter.MeterValues{EgressGiB: 1}), 1e-12)
		require.InDelta(t, 0.000006, PriceCents(billingmeter.MeterValues{DiskGiBSeconds: 1}), 1e-12)
		require.InDelta(t, 0.2, PriceCents(billingmeter.MeterValues{ActiveKeys: 1}), 1e-12)
	})

	t.Run("meters sum", func(t *testing.T) {
		// $0.50 plan-month of egress (10 GiB) plus 100 active keys ($0.20).
		got := PriceCents(billingmeter.MeterValues{EgressGiB: 10, ActiveKeys: 100})
		require.InDelta(t, 10*5.0+100*0.2, got, 1e-9)
	})
}
