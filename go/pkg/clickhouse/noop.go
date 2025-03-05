package clickhouse

import (
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// noop implements the Bufferer interface but discards all events.
// This is useful for testing or when ClickHouse functionality is not needed,
// such as in development environments or when running integration tests.
type noop struct{}

var _ Bufferer = &noop{}

// BufferApiRequest implements the Bufferer interface but discards the event.
func (n *noop) BufferApiRequest(schema.ApiRequestV1) {
	// Intentionally empty - discards the event
}

// BufferKeyVerification implements the Bufferer interface but discards the event.
func (n *noop) BufferKeyVerification(schema.KeyVerificationRequestV1) {
	// Intentionally empty - discards the event
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
