package billingreconcile

import (
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/billingperiod"
)

// Only cycle invoices are existence candidates; prorations share the window.
const billingReasonSubscriptionCycle = "subscription_cycle"

// findExistence returns the one finalized cycle invoice belonging to the period,
// or nil + a structural finding (missing/duplicate/misaligned/not-finalized).
//
// A cycle invoice belongs to the month when its period ENDS on the calendar-month
// boundary; the start may be later (a mid-month subscribe's first cycle is
// [subscribe_date, 1st]), so only period_end is checked. Half-open overlap keeps
// the previous cycle (ending at this period's start) out of the candidate set.
func findExistence(candidates []InvoiceCandidate, p billingperiod.Period) (*InvoiceCandidate, []Finding) {
	wantStart := p.Start().Unix()
	wantEnd := p.End().Unix()

	var matches, misaligned []InvoiceCandidate
	var anyDraft bool
	for _, c := range candidates {
		if c.BillingReason != billingReasonSubscriptionCycle {
			continue // prorations / create invoices are not cycle candidates
		}
		if c.PeriodEnd <= wantStart || c.PeriodStart >= wantEnd {
			continue // does not overlap the period (half-open)
		}
		if !c.Status.Finalized() {
			if c.Status == InvoiceStatusDraft {
				anyDraft = true
			}
			continue
		}
		if c.PeriodEnd == wantEnd && c.PeriodStart >= wantStart {
			matches = append(matches, c)
		} else {
			misaligned = append(misaligned, c)
		}
	}

	switch {
	case len(matches) == 1:
		return &matches[0], nil
	case len(matches) > 1:
		return nil, []Finding{structuralExistence(fmt.Sprintf(
			"%d finalized subscription_cycle invoices ending on the %s boundary: %s",
			len(matches), p.Key(), invoiceIDs(matches)))}
	case len(misaligned) > 0:
		return nil, []Finding{structuralExistence(fmt.Sprintf(
			"subscription_cycle invoice %s for %s does not end on the calendar-month boundary (or starts before it): [%s, %s]",
			misaligned[0].ID, p.Key(),
			time.Unix(misaligned[0].PeriodStart, 0).UTC().Format(time.RFC3339),
			time.Unix(misaligned[0].PeriodEnd, 0).UTC().Format(time.RFC3339)))}
	case anyDraft:
		return nil, []Finding{structuralExistence(fmt.Sprintf(
			"subscription_cycle invoice draft found for %s but never finalized", p.Key()))}
	default:
		return nil, []Finding{structuralExistence(fmt.Sprintf(
			"no subscription_cycle invoice for %s", p.Key()))}
	}
}

func structuralExistence(detail string) Finding {
	return Finding{Check: CheckExistence, Class: VerdictStructural, Meter: "", DriftCents: 0, Detail: detail}
}

func invoiceIDs(cs []InvoiceCandidate) string {
	ids := make([]string, len(cs))
	for i, c := range cs {
		ids[i] = c.ID
	}
	return strings.Join(ids, ", ")
}
