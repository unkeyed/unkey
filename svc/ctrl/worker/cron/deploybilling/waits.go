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

// DefaultFinalizeDelay is the production wait between the close's final meter
// push and invoice finalization, giving Stripe's asynchronous meter
// aggregation time to fold the final push into the draft's lines. Finalizing
// too early silently locks in the previous hourly value, so this is sized to
// Stripe's own ~1h draft-settling window; the budget is large (finalize at
// period end +25h vs the +48h backstop). See waitForMeterAggregation.
const DefaultFinalizeDelay = time.Hour

// waitForUsageIngestion blocks until late ClickHouse rows for the closed period
// are likely ingested. Both close paths (HandleClose and HandleCloseWorkspace)
// call this before the final usage read. Skipped when wall clock is still
// inside the period (CloseAllowed only via a Stripe period-end hint, e.g. test
// clocks ahead of wall clock).
func waitForUsageIngestion(ctx restate.ObjectContext, p billingperiod.Period, now time.Time) error {
	if now.Before(p.End()) {
		return nil
	}

	ingestSafe := p.End().Add(usageIngestLateness)
	if now.Before(ingestSafe) {
		if err := restate.Sleep(ctx, ingestSafe.Sub(now)); err != nil {
			return fmt.Errorf("wait for usage ingestion: %w", err)
		}
	}

	return nil
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
