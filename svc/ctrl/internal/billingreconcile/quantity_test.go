package billingreconcile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnderbillThresholdCents(t *testing.T) {
	require.Equal(t, int64(50), underbillThresholdCents(0))
	require.Equal(t, int64(50), underbillThresholdCents(1_000))    // 0.5% of $10 = 5c, floor wins
	require.Equal(t, int64(500), underbillThresholdCents(100_000)) // 0.5% of $1,000 = $5, percentage wins
}

// oneMeter builds a single-meter metered map billing invoicedQty, and the live
// map recording liveQty, for the cpu_seconds meter.
func oneMeter(t *testing.T, m Meter, invoicedQty, liveQty float64) (map[Meter]InvoiceLine, map[Meter]float64) {
	t.Helper()
	line, _ := meterLineFor(m, invoicedQty)
	live := map[Meter]float64{}
	for _, mm := range Meters() {
		live[mm] = 0
	}
	live[m] = liveQty
	return map[Meter]InvoiceLine{m: line}, live
}

func TestCheckQuantities(t *testing.T) {
	m := MeterCPUSeconds

	t.Run("exact match: no finding", func(t *testing.T) {
		metered, live := oneMeter(t, m, 1_000_000, 1_000_000)
		require.Empty(t, checkQuantities(metered, live))
	})

	t.Run("CH higher under the bar: clean", func(t *testing.T) {
		// +100 CPU-seconds prices to ~0.07 cents, far under the $0.50 floor.
		metered, live := oneMeter(t, m, 1_000_000, 1_000_100)
		require.Empty(t, checkQuantities(metered, live))
	})

	t.Run("CH higher over the bar: late_data_underbill", func(t *testing.T) {
		metered, live := oneMeter(t, m, 1_000_000, 2_000_000)
		findings := checkQuantities(metered, live)
		require.Len(t, findings, 1)
		require.Equal(t, VerdictLateDataUnderbill, findings[0].Class)
		require.Less(t, findings[0].DriftCents, int64(0))
	})

	t.Run("invoice higher over one cent: overbill, no dollar floor", func(t *testing.T) {
		metered, live := oneMeter(t, m, 1_000_000, 10_000)
		findings := checkQuantities(metered, live)
		require.Len(t, findings, 1)
		require.Equal(t, VerdictOverbill, findings[0].Class)
		require.Greater(t, findings[0].DriftCents, int64(0))
	})

	t.Run("invoice higher within one cent of rounding noise: clean", func(t *testing.T) {
		// A one-unit gap prices to ~0.0007 cents: rounds to zero, under the 1c
		// tolerance.
		metered, live := oneMeter(t, m, 1_000_000, 999_999)
		require.Empty(t, checkQuantities(metered, live))
	})

	t.Run("missing line but material live usage: late_data_underbill on the floor", func(t *testing.T) {
		live := map[Meter]float64{}
		for _, mm := range Meters() {
			live[mm] = 0
		}
		live[MeterActiveKeys] = 300 // 60 cents, past the $0.50 floor with no line
		findings := checkQuantities(map[Meter]InvoiceLine{}, live)
		require.Len(t, findings, 1)
		require.Equal(t, MeterActiveKeys, findings[0].Meter)
		require.Equal(t, VerdictLateDataUnderbill, findings[0].Class)
		require.Equal(t, int64(-60), findings[0].DriftCents)
	})

	t.Run("float noise below a cent is not an overbill", func(t *testing.T) {
		qty := 1_000_000 * (1 + 1e-12)
		metered, live := oneMeter(t, m, qty, 1_000_000)
		require.Empty(t, checkQuantities(metered, live))
	})
}
