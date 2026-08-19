package deploybilling

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/billingperiod"
)

// usageIngestLateness is how long the close waits after period end before the
// final usage read, so late ClickHouse rows (buffered flushes, ingestion
// backpressure during an outage) are included in the closed invoice. The
// invoice.created webhook already claimed the draft (backstop finalization
// scheduled at period end plus 48h), so Stripe cannot finalize underneath the
// wait, and the final read happens after the sleep, so anything ingested in
// the meantime is picked up. Sized to the top of the industry's 12-24h
// grace-window range so even a day-long ingestion outage lands inside the
// closed invoice.
const usageIngestLateness = 24 * time.Hour

// simulatedClockMinimumLead distinguishes a Stripe test clock from ordinary
// production clock skew. Production Stripe cannot roll a calendar-month
// renewal an hour before that month ends; test-clock periods can be weeks ahead.
const simulatedClockMinimumLead = time.Hour

// DefaultFinalizeDelay is the production wait between the close's final meter
// push and invoice finalization, giving Stripe's asynchronous meter
// aggregation time to fold the final push into the draft's lines. Finalizing
// too early silently locks in the previous hourly value, so this is sized to
// Stripe's own ~1h draft-settling window; the budget is large (finalize at
// period end +25h vs the +48h backstop). See waitForMeterAggregation.
const DefaultFinalizeDelay = time.Hour

// waitForUsageIngestion blocks until late ClickHouse rows for the closed period
// are likely ingested. Both close paths (HandleClose and HandleCloseWorkspace)
// call this before the final usage read. Skipped only when the period is far
// enough ahead of wall time to prove a Stripe test clock is in use; a small
// future skew still waits through period end and the full ingestion window.
func waitForUsageIngestion(ctx restate.ObjectContext, p billingperiod.Period, now time.Time) error {
	delay := usageIngestionDelay(p, now)
	if delay <= 0 {
		return nil
	}

	if err := restate.Sleep(ctx, delay); err != nil {
		return fmt.Errorf("wait for usage ingestion: %w", err)
	}

	return nil
}

func usageIngestionDelay(p billingperiod.Period, now time.Time) time.Duration {
	periodEnd := p.End()
	if periodEnd.Sub(now) > simulatedClockMinimumLead {
		return 0
	}

	ingestSafe := periodEnd.Add(usageIngestLateness)
	if !now.Before(ingestSafe) {
		return 0
	}
	return ingestSafe.Sub(now)
}

// waitForMeterAggregation blocks until Stripe can fold the final meter push into
// the draft invoice's lines. Skipped when no push ran or FinalizeDelay is zero.
func (h *Handler) waitForMeterAggregation(ctx restate.ObjectContext, pushed bool) error {
	if !pushed || h.finalizeDelay <= 0 {
		return nil
	}

	if err := restate.Sleep(ctx, h.finalizeDelay); err != nil {
		return fmt.Errorf("wait for meter aggregation: %w", err)
	}

	return nil
}
