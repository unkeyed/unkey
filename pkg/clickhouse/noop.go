package clickhouse

import (
	"context"
	"time"

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

// GetVerificationsByExternalIDPerKey implements the Querier interface but always returns nil.
func (n *noop) GetVerificationsByExternalIDPerKey(ctx context.Context, req VerificationTimeseriesPerKeyRequest) ([]VerificationTimeseriesPerKey, error) {
	return nil, nil
}

// GetBillableUsageAboveThreshold implements the Querier interface but always returns an empty map.
func (n *noop) GetBillableUsageAboveThreshold(ctx context.Context, year, month int, minUsage int64) (map[string]int64, error) {
	return make(map[string]int64), nil
}

// GetInstanceMeterUsage implements the Querier interface but always returns an empty slice.
func (n *noop) GetInstanceMeterUsage(ctx context.Context, req GetInstanceMeterUsageRequest) ([]InstanceMeterUsage, error) {
	return nil, nil
}

// GetComputeUsageByEnvironment implements the Querier interface but always returns an empty slice
func (n *noop) GetComputeUsageByEnvironment(ctx context.Context, workspaceID string, start, end time.Time) ([]ComputeUsageByEnvironment, error) {
	return nil, nil
}

// GetActiveKeysByApp implements the Querier interface but always returns an empty slice
func (n *noop) GetActiveKeysByApp(ctx context.Context, workspaceID string, year, month int) ([]ActiveKeysByApp, error) {
	return nil, nil
}

// GetDeploymentRequestCount implements the Querier interface but always returns 0.
func (n *noop) GetDeploymentRequestCount(ctx context.Context, req GetDeploymentRequestCountRequest) (int64, error) {
	return 0, nil
}

// GetKeyLastUsedBatchPartitioned implements the Querier interface but always returns an empty slice.
func (n *noop) GetKeyLastUsedBatchPartitioned(ctx context.Context, req GetKeyLastUsedBatchRequest) ([]KeyLastUsed, error) {
	return nil, nil
}

// GetBuildLogs implements the Querier interface but always returns no entries
func (n *noop) GetBuildLogs(ctx context.Context, req GetBuildLogsRequest) (BuildLogsPage, error) {
	return BuildLogsPage{Entries: []BuildLogEntry{}, HasMore: false}, nil
}

// ListApiRequests implements the Querier interface but always returns no rows
func (n *noop) ListApiRequests(ctx context.Context, req ListApiRequestsRequest) ([]ApiRequestLog, error) {
	return []ApiRequestLog{}, nil
}

// GetApiRequestMetrics implements the Querier interface but always returns no points
func (n *noop) GetApiRequestMetrics(ctx context.Context, req ApiRequestMetricsRequest) ([]ApiRequestMetricsPoint, error) {
	return []ApiRequestMetricsPoint{}, nil
}

// InsertAuditLogs implements the Querier interface but discards the input.
func (n *noop) InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error {
	return nil
}

// EraseWorkspaceRows implements the Querier interface but finds no rows.
func (n *noop) EraseWorkspaceRows(ctx context.Context, table, workspaceID string) (int64, error) {
	return 0, nil
}

// EraseEndUserRows implements the Querier interface but finds no rows.
func (n *noop) EraseEndUserRows(ctx context.Context, table string, endUser EndUserErasure) (int64, error) {
	return 0, nil
}

// PseudonymizeAuditActor implements the Querier interface but finds no rows.
func (n *noop) PseudonymizeAuditActor(ctx context.Context, actorID string) (int64, error) {
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

// NewNoop creates a new no-op implementation of the ClickHouse interface.
// This implementation discards all operations without processing them.
//
// For no-op batch processors, use batch.NewNoop[T]() instead.
func NewNoop() *noop {
	return &noop{}
}
