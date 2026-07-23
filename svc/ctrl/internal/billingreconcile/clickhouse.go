package billingreconcile

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// clickhouseUsageReader runs the same two queries and aggregation the hourly
// push bills from (deploybilling.AggregateUsage / MergeActiveKeys), scoped to
// one workspace. Plain context.Context: a reconcile is read-only and idempotent,
// so there is nothing to journal.
type clickhouseUsageReader struct {
	usage deploybilling.UsageReader
}

var _ UsageReader = (*clickhouseUsageReader)(nil)

// NewClickHouseUsageReader builds a UsageReader from any
// deploybilling.UsageReader (in production, *clickhouse.Client).
func NewClickHouseUsageReader(usage deploybilling.UsageReader) UsageReader {
	return &clickhouseUsageReader{usage: usage}
}

func (r *clickhouseUsageReader) WorkspaceUsage(
	ctx context.Context,
	workspaceID string,
	start, end time.Time,
) (billingmeter.MeterValues, error) {
	rows, err := r.usage.GetInstanceMeterUsage(ctx, clickhouse.GetInstanceMeterUsageRequest{
		WorkspaceID:  workspaceID,
		WorkspaceIDs: nil,
		Start:        start.UnixMilli(),
		End:          end.UnixMilli(),
	})
	if err != nil {
		return billingmeter.MeterValues{}, fault.Wrap(err, fault.Internal("query instance usage for "+workspaceID)) //nolint:exhaustruct // zero-value return on the error path
	}

	// Active keys are counted per calendar month; the window's month is the
	// month it starts in (a first partial cycle sits inside one month).
	keyRows, err := r.usage.GetActiveKeysUsage(ctx, clickhouse.GetActiveKeysUsageRequest{
		WorkspaceID:  workspaceID,
		WorkspaceIDs: nil,
		Year:         start.Year(),
		Month:        start.Month(),
	})
	if err != nil {
		return billingmeter.MeterValues{}, fault.Wrap(err, fault.Internal("query active keys for "+workspaceID)) //nolint:exhaustruct // zero-value return on the error path
	}

	values := deploybilling.AggregateUsage(rows)
	deploybilling.MergeActiveKeys(values, keyRows)
	// Zero value when the workspace had no usage at all this period: a
	// scaled-to-zero deployment with no key verifications is a legitimate,
	// clean state, not a missing-data error.
	return values[workspaceID], nil
}
