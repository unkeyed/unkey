package billingreconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// TestCatalogRatesArePinned asserts the reconcile engine prices against the
// exact rates deploybilling bills with, so the two never drift.
func TestCatalogRatesArePinned(t *testing.T) {
	require.Equal(t, deploybilling.CentsPerCPUSecond, rateCents(MeterCPUSeconds))
	require.Equal(t, deploybilling.CentsPerMemoryGiBSecond, rateCents(MeterMemoryGiBSeconds))
	require.Equal(t, deploybilling.CentsPerEgressGiB, rateCents(MeterEgressGiB))
	require.Equal(t, deploybilling.CentsPerDiskGiBSecond, rateCents(MeterDiskGiBSeconds))
	require.Equal(t, deploybilling.CentsPerActiveKey, rateCents(MeterActiveKeys))
}

func TestRatesEqual(t *testing.T) {
	require.True(t, ratesEqual(0.0006944, 0.0006944))
	require.True(t, ratesEqual(5.0, 5.0*(1+1e-10)))
	require.False(t, ratesEqual(5.0, 5.5))
}

// checkPricesFor runs the price check for a single metered line against a price
// catalog.
func checkPricesFor(t *testing.T, m Meter, line InvoiceLine, catalog map[string]Price) ([]Finding, error) {
	t.Helper()
	r := New(nil, &fakePrices{byLookupKey: catalog, err: nil}, nil)
	return r.checkPrices(context.Background(), map[Meter]InvoiceLine{m: line})
}

func TestCheckPrices(t *testing.T) {
	m := MeterCPUSeconds

	t.Run("clean: catalog and line rate match, amount recomputes", func(t *testing.T) {
		line, price := meterLineFor(m, 1_000_000)
		findings, err := checkPricesFor(t, m, line, map[string]Price{m.LookupKey(): price})
		require.NoError(t, err)
		require.Empty(t, findings)
	})

	t.Run("stripe catalog rate diverges from pinned: structural", func(t *testing.T) {
		line, price := meterLineFor(m, 1_000_000)
		price.UnitAmountDecimal *= 2
		findings, err := checkPricesFor(t, m, line, map[string]Price{m.LookupKey(): price})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "stripe rate")
	})

	t.Run("line's own billed rate diverges from pinned: structural", func(t *testing.T) {
		line, price := meterLineFor(m, 1_000_000)
		line.UnitAmountDecimal *= 2 // the line billed a stale rate
		findings, err := checkPricesFor(t, m, line, map[string]Price{m.LookupKey(): price})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "line rate")
	})

	t.Run("reprice: lookup key resolves to a different price id: structural", func(t *testing.T) {
		line, price := meterLineFor(m, 1_000_000)
		price.ID = "price_cpu_v2"
		findings, err := checkPricesFor(t, m, line, map[string]Price{m.LookupKey(): price})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "resolves to price_cpu_v2")
	})

	t.Run("recomputed line amount off by two cents: structural", func(t *testing.T) {
		line, price := meterLineFor(m, 1_000_000)
		line.AmountCents += 2
		findings, err := checkPricesFor(t, m, line, map[string]Price{m.LookupKey(): price})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "recomputes to")
	})

	t.Run("recomputed line amount off by one cent: within rounding tolerance", func(t *testing.T) {
		line, price := meterLineFor(m, 1_000_000)
		line.AmountCents++
		findings, err := checkPricesFor(t, m, line, map[string]Price{m.LookupKey(): price})
		require.NoError(t, err)
		require.Empty(t, findings)
	})

	t.Run("no price for the lookup key: structural, not an error", func(t *testing.T) {
		line, _ := meterLineFor(m, 1_000_000)
		findings, err := checkPricesFor(t, m, line, map[string]Price{})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "no stripe price")
	})

	t.Run("a hard read failure propagates", func(t *testing.T) {
		line, _ := meterLineFor(m, 1_000_000)
		wantErr := errors.New("stripe is down")
		r := New(nil, &fakePrices{byLookupKey: nil, err: wantErr}, nil)
		_, err := r.checkPrices(context.Background(), map[Meter]InvoiceLine{m: line})
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("meters with no billed line are skipped", func(t *testing.T) {
		r := New(nil, &fakePrices{byLookupKey: map[string]Price{}, err: nil}, nil)
		findings, err := r.checkPrices(context.Background(), map[Meter]InvoiceLine{})
		require.NoError(t, err)
		require.Empty(t, findings)
	})
}
