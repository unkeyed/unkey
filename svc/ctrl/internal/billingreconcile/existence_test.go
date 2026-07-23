package billingreconcile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindExistence(t *testing.T) {
	p := fixturePeriod(t)
	start := p.Start().Unix()
	end := p.End().Unix()
	priorEnd := start // the previous cycle ends exactly at this period's start

	cyc := func(id string, status InvoiceStatus, ps, pe int64) InvoiceCandidate {
		return InvoiceCandidate{ID: id, Status: status, BillingReason: "subscription_cycle", PeriodStart: ps, PeriodEnd: pe}
	}

	t.Run("ok: exactly one finalized calendar-month invoice", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_prev", InvoiceStatusPaid, p.Prev().Start().Unix(), priorEnd),
			cyc("in_1", InvoiceStatusPaid, start, end),
		}, p)
		require.NotNil(t, matched)
		require.Equal(t, "in_1", matched.ID)
		require.Empty(t, findings)
	})

	t.Run("ok: first partial cycle after a mid-month subscribe (end aligned, start later)", func(t *testing.T) {
		// A subscription created mid-month bills [subscribe_date, 1st] as its
		// first cycle: period_end on the boundary, period_start well after the
		// month start. Legitimate, must match (and drive usage from that window).
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_partial", InvoiceStatusPaid, start+15*86400, end),
		}, p)
		require.NotNil(t, matched)
		require.Equal(t, "in_partial", matched.ID)
		require.Equal(t, start+15*86400, matched.PeriodStart)
		require.Empty(t, findings)
	})

	t.Run("missing: no candidates", func(t *testing.T) {
		matched, findings := findExistence(nil, p)
		require.Nil(t, matched)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "no subscription_cycle invoice")
	})

	t.Run("missing: only the previous cycle, excluded by half-open overlap", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_prev", InvoiceStatusPaid, p.Prev().Start().Unix(), priorEnd),
		}, p)
		require.Nil(t, matched)
		require.Contains(t, findings[0].Detail, "no subscription_cycle invoice")
	})

	t.Run("missing: only void noise", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_void", InvoiceStatusVoid, start, end),
		}, p)
		require.Nil(t, matched)
		require.Contains(t, findings[0].Detail, "no subscription_cycle invoice")
	})

	t.Run("duplicate: two finalized aligned invoices", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_1", InvoiceStatusPaid, start, end),
			cyc("in_2", InvoiceStatusOpen, start, end),
		}, p)
		require.Nil(t, matched)
		require.Contains(t, findings[0].Detail, "2 finalized")
		require.Contains(t, findings[0].Detail, "in_1")
		require.Contains(t, findings[0].Detail, "in_2")
	})

	t.Run("misaligned: end off the calendar-month boundary (anchor collapse)", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_1", InvoiceStatusPaid, start-86400, end-86400),
		}, p)
		require.Nil(t, matched)
		require.Contains(t, findings[0].Detail, "does not end on the calendar-month boundary")
	})

	t.Run("not_finalized: only a draft for the period", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_1", InvoiceStatusDraft, start, end),
		}, p)
		require.Nil(t, matched)
		require.Contains(t, findings[0].Detail, "never finalized")
	})

	t.Run("ok: a leftover draft alongside the finalized invoice does not block", func(t *testing.T) {
		matched, findings := findExistence([]InvoiceCandidate{
			cyc("in_draft", InvoiceStatusDraft, start, end),
			cyc("in_1", InvoiceStatusPaid, start, end),
		}, p)
		require.NotNil(t, matched)
		require.Equal(t, "in_1", matched.ID)
		require.Empty(t, findings)
	})
}
