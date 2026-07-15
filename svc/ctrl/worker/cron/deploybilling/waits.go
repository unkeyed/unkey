package deploybilling

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/billingperiod"
)

const usageIngestLateness = 15 * time.Minute

// DefaultFinalizeDelay is the production wait between the close's final meter
// push and invoice finalization. See waitForMeterAggregation.
const DefaultFinalizeDelay = 20 * time.Minute

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
