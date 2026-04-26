package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// noop implements the ClickHouse interface but discards all operations.
// This is useful for testing or when ClickHouse functionality is not needed,
// such as in development environments or when running integration tests.
type noop struct{}

var _ ClickHouse = (*noop)(nil)

// GetBillableVerifications implements the Querier interface but always returns 0.
func (n *noop) GetBillableVerifications(ctx context.Context, workspaceID string, year, month int) (int64, error) {
	return 0, nil
}

// GetBillableRatelimits implements the Querier interface but always returns 0.
func (n *noop) GetBillableRatelimits(ctx context.Context, workspaceID string, year, month int) (int64, error) {
	return 0, nil
}

// GetBillableUsageAboveThreshold implements the Querier interface but always returns an empty map.
func (n *noop) GetBillableUsageAboveThreshold(ctx context.Context, year, month int, minUsage int64) (map[string]int64, error) {
	return make(map[string]int64), nil
}

// GetDeploymentRequestCount implements the Querier interface but always returns 0.
func (n *noop) GetDeploymentRequestCount(ctx context.Context, req GetDeploymentRequestCountRequest) (int64, error) {
	return 0, nil
}

// GetKeyLastUsedBatchPartitioned implements the Querier interface but always returns an empty slice.
func (n *noop) GetKeyLastUsedBatchPartitioned(ctx context.Context, req GetKeyLastUsedBatchRequest) ([]KeyLastUsed, error) {
	return nil, nil
}

// InsertAuditLogs implements the Querier interface but discards the input.
func (n *noop) InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error {
	return nil
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

// NewNoop creates a new no-op implementation of the ClickHouse interface.
// This implementation discards all operations without processing them.
//
// For no-op batch processors, use batch.NewNoop[T]() instead.
func NewNoop() *noop {
	return &noop{}
}
