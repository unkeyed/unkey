package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// noop implements the Bufferer interface but discards all events.
// This is useful for testing or when ClickHouse functionality is not needed,
// such as in development environments or when running integration tests.
type noop struct{}

var _ Bufferer = (*noop)(nil)
var _ Bufferer = (*noop)(nil)

func (n *noop) BufferApiRequest(schema.ApiRequestV2) {
	// Intentionally empty - discards the event
}

// BufferKeyVerificationV2 implements the Bufferer interface but discards the event.
func (n *noop) BufferKeyVerificationV2(schema.KeyVerificationV2) {
	// Intentionally empty - discards the event
}

// BufferRatelimit implements the Bufferer interface but discards the event.
func (n *noop) BufferRatelimit(req schema.RatelimitV2) {
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

func (n *noop) Conn() ch.Conn {
	return nil
}

// QueryToMaps implements the Querier interface but always returns an empty slice.
func (n *noop) QueryToMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

// Exec implements the Querier interface but does nothing.
func (n *noop) Exec(ctx context.Context, sql string, args ...any) error {
	return nil
}

// ConfigureUser implements the Querier interface but does nothing.
func (n *noop) ConfigureUser(ctx context.Context, config UserConfig) error {
	return nil
}

func (n *noop) Ping(ctx context.Context) error {
	return nil
}

// Close closes the underlying ClickHouse connection.
func (n *noop) Close() error {
	return nil
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
