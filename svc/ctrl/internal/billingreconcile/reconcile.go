package billingreconcile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/fault"
)

// The candidate creation window: [period.Start-48h, period.End+14d]. Lookback
// reaches the prior renewal (funds this period's credit); lookahead covers the
// cycle invoice plus headroom past the 48h finalization backstop.
const (
	invoiceWindowBefore = 48 * time.Hour
	invoiceWindowAfter  = 14 * 24 * time.Hour
)

// ReconcileWorkspace compares ws's finalized Deploy invoice for period p against
// live ClickHouse usage and returns a verdict plus its findings.
//
// A read error other than ErrNotFound (Stripe/ClickHouse outage) aborts with an
// error, never a Result and never VerdictClean, so a cron retries rather than
// recording a false clean. A business-meaningful ErrNotFound (missing catalog
// price) folds into the Result as the structural finding it represents.
func (r *Reconciler) ReconcileWorkspace(ctx context.Context, ws WorkspaceRef, p billingperiod.Period) (Result, error) {
	result := Result{
		WorkspaceID: ws.WorkspaceID,
		Period:      p,
		InvoiceID:   "",
		Verdict:     VerdictClean,
		Findings:    nil,
	}
	if ws.WorkspaceID == "" || ws.StripeSubscriptionID == "" {
		return result, fault.New("reconcile requires a workspace id and a stripe subscription id")
	}

	from := p.Start().Add(-invoiceWindowBefore)
	to := p.End().Add(invoiceWindowAfter)
	candidates, err := r.invoices.ListInvoices(ctx, ws.StripeSubscriptionID, from, to)
	if err != nil {
		return result, fault.Wrap(err, fault.Internal("list subscription_cycle invoices"))
	}

	matched, existenceFindings := findExistence(candidates, p)
	if matched == nil {
		// Existence failure short-circuits: without exactly one finalized
		// invoice for the period there is nothing to compare a line, a credit,
		// or a total against.
		result.Findings = existenceFindings
		result.Verdict = foldVerdict(result.Findings)
		return result, nil
	}
	result.InvoiceID = matched.ID

	inv, err := r.invoices.GetInvoice(ctx, matched.ID)
	if err != nil {
		return result, fault.Wrap(err, fault.Internal("get invoice "+matched.ID))
	}

	// Usage over the invoice's actual window, not the calendar month, so a first
	// partial cycle compares against matching usage (identical for a full cycle).
	live, err := r.usage.WorkspaceUsage(ctx, ws.WorkspaceID,
		time.Unix(matched.PeriodStart, 0), time.Unix(matched.PeriodEnd, 0))
	if err != nil {
		return result, fault.Wrap(err, fault.Internal("re-derive clickhouse usage"))
	}

	metered, lineFindings := classifyLines(inv)
	result.Findings = append(result.Findings, lineFindings...)

	result.Findings = append(result.Findings, checkQuantities(metered, meterQuantities(live))...)

	priceFindings, err := r.checkPrices(ctx, metered)
	if err != nil {
		return result, err
	}
	result.Findings = append(result.Findings, priceFindings...)

	grant, hasPrevious, err := r.expectedCreditGrant(ctx, candidates, matched)
	if err != nil {
		return result, err
	}
	result.Findings = append(result.Findings, checkCredit(inv, metered, grant, hasPrevious)...)

	result.Findings = append(result.Findings, checkTotal(inv)...)

	result.Verdict = foldVerdict(result.Findings)
	return result, nil
}

// foldVerdict returns the most severe class among the findings (structural >
// overbill > late_data_underbill > clean), clean when there are none.
func foldVerdict(findings []Finding) Verdict {
	verdict := VerdictClean
	for _, f := range findings {
		verdict = worse(verdict, f.Class)
	}
	return verdict
}

// worse returns the more severe of two verdicts.
func worse(a, b Verdict) Verdict {
	if b.severity() > a.severity() {
		return b
	}
	return a
}

// classifyLines maps the invoice's metered lines by meter. Plan-fee lines are
// skipped (checked when their period reconciles); an unrecognized lookup_key is
// structural; a duplicate meter line is structural and drops that meter.
func classifyLines(inv Invoice) (map[Meter]InvoiceLine, []Finding) {
	var findings []Finding
	metered := make(map[Meter]InvoiceLine)
	duplicated := make(map[Meter]bool)

	known := make(map[string]Meter, len(Meters()))
	for _, m := range Meters() {
		known[m.LookupKey()] = m
	}

	for _, line := range inv.Lines {
		switch {
		case known[line.PriceLookupKey] != "":
			m := known[line.PriceLookupKey]
			if _, dup := metered[m]; dup || duplicated[m] {
				duplicated[m] = true
				delete(metered, m)
				findings = append(findings, Finding{
					Check:      CheckExistence,
					Class:      VerdictStructural,
					Meter:      m,
					DriftCents: 0,
					Detail:     fmt.Sprintf("invoice %s has multiple lines for meter %s", inv.ID, m),
				})
				continue
			}
			metered[m] = line
		case strings.HasPrefix(line.PriceLookupKey, planLookupPrefix):
			// Plan fee; not a per-meter check here.
		default:
			findings = append(findings, Finding{
				Check:      CheckExistence,
				Class:      VerdictStructural,
				Meter:      "",
				DriftCents: 0,
				Detail: fmt.Sprintf("invoice %s line %s bills through unrecognized price %q (lookup key %q)",
					inv.ID, line.ID, line.PriceID, line.PriceLookupKey),
			})
		}
	}
	return metered, findings
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
