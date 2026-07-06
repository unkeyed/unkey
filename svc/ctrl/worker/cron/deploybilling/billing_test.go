package deploybilling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
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

		out := aggregateUsage(rows)
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
		require.Empty(t, aggregateUsage(nil))
	})
}

func TestClosedPeriodToBill(t *testing.T) {
	utc := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
	}

	t.Run("bills the month before the open period, never the open one", func(t *testing.T) {
		p, err := billingperiod.Parse("2026-07")
		require.NoError(t, err)
		year, month, ready := closedPeriodToBill(p, utc(2026, time.July, 15))
		require.Equal(t, 2026, year)
		require.Equal(t, int(time.June), month, "the open month (July) is never billed; its predecessor is")
		require.True(t, ready)
	})

	t.Run("crosses the year boundary", func(t *testing.T) {
		p, err := billingperiod.Parse("2026-01")
		require.NoError(t, err)
		year, month, _ := closedPeriodToBill(p, utc(2026, time.January, 15))
		require.Equal(t, 2025, year)
		require.Equal(t, int(time.December), month)
	})

	t.Run("defers until the settle window elapses", func(t *testing.T) {
		p, err := billingperiod.Parse("2026-07")
		require.NoError(t, err)
		// First day of the new month: the just-closed month may still be
		// receiving late-arriving events, so billing is deferred.
		_, _, ready := closedPeriodToBill(p, utc(2026, time.July, 1))
		require.False(t, ready, "day 1 is inside the settle window")
		_, _, ready = closedPeriodToBill(p, utc(2026, time.July, endUserBillingSettleDays))
		require.True(t, ready, "billing is ready once the settle window has elapsed")
	})
}
