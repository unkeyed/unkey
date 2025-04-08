package clickhouse

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// noop implements the Bufferer interface but discards all events.
// This is useful for testing or when ClickHouse functionality is not needed,
// such as in development environments or when running integration tests.
type noop struct{}

var _ Bufferer = (*noop)(nil)
var _ Bufferer = (*noop)(nil)

// BufferApiRequest implements the Bufferer interface but discards the event.
func (n *noop) BufferApiRequest(schema.ApiRequestV1) {
	// Intentionally empty - discards the event
}

// BufferKeyVerification implements the Bufferer interface but discards the event.
func (n *noop) BufferKeyVerification(schema.KeyVerificationRequestV1) {
	// Intentionally empty - discards the event
}

// BufferRatelimit implements the Bufferer interface but discards the event.
func (n *noop) BufferRatelimit(req schema.RatelimitRequestV1) {
	// Intentionally empty - discards the event
}

// GetBillableVerifications implements the Bufferer interface but always returns 0.
func (n *noop) GetBillableVerifications(ctx context.Context, workspaceID string, year, month int) (int64, error) {
	return 0, nil
}

// GetBillableRatelimits implements the Bufferer interface but always returns 0.
func (n *noop) GetBillableRatelimits(ctx context.Context, workspaceID string, year, month int) (int64, error) {
	return 0, nil
}

// NewNoop creates a new no-op implementation of the Bufferer interface.
// This implementation simply discards all events without processing them.
//
// This is useful for:
// - Development environments where ClickHouse is not available
// - Testing where analytics are not relevant
// - Scenarios where analytics are optional and not configured
//
// Example:
//
//	var bufferer clickhouse.Bufferer
//	if config.ClickhouseURL != "" {
//	    ch, err := clickhouse.New(clickhouse.Config{
//	        URL:    config.ClickhouseURL,
//	        Logger: logger,
//	    })
//	    if err != nil {
//	        logger.Warn("Failed to initialize ClickHouse, analytics will be disabled")
//	        bufferer = clickhouse.NewNoop()
//	    } else {
//	        bufferer = ch
//	    }
//	} else {
//	    bufferer = clickhouse.NewNoop()
//	}
func NewNoop() *noop {
	return &noop{}
}
