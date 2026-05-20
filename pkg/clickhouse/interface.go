package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

type Querier interface {
	// Conn returns a connection to the ClickHouse database.
	Conn() ch.Conn

	// QueryToMaps executes a query and scans all rows into a slice of maps.
	// Each map represents a row with column names as keys.
	// This is useful for dynamic queries where the schema is not known at compile time.
	QueryToMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error)

	// Exec executes a DDL or DML statement (CREATE, ALTER, DROP, etc.)
	Exec(ctx context.Context, sql string, args ...any) error

	// ConfigureUser creates or updates a ClickHouse user with permissions, quotas, and settings.
	// This is idempotent and can be called multiple times to update configuration.
	ConfigureUser(ctx context.Context, config UserConfig) error

	GetBillableVerifications(ctx context.Context, workspaceID string, year, month int) (int64, error)

	GetBillableRatelimits(ctx context.Context, workspaceID string, year, month int) (int64, error)

	// GetBillableUsageAboveThreshold returns total billable usage for workspaces that exceed a minimum threshold.
	// This pre-filters in ClickHouse rather than returning all workspaces, making it efficient for quota checking.
	// Returns a map from workspace ID to total usage count (only for workspaces >= minUsage).
	GetBillableUsageAboveThreshold(ctx context.Context, year, month int, minUsage int64) (map[string]int64, error)

	// GetDeploymentRequestCount returns the number of sentinel requests routed to a
	// deployment within a recent time window, used to detect idle deployments for scale-down.
	// Returns 0 (not an error) when the deployment has received no traffic.
	GetDeploymentRequestCount(ctx context.Context, req GetDeploymentRequestCountRequest) (int64, error)

	// GetKeyLastUsedBatchPartitioned returns keys in a specific hash partition
	// (cityHash64(key_id) % totalPartitions == partition) after the given cursor,
	// ordered by (time, key_id). Used by the KeyLastUsedSync partition workers.
	GetKeyLastUsedBatchPartitioned(ctx context.Context, req GetKeyLastUsedBatchRequest) ([]KeyLastUsed, error)

	// InsertAuditLogs synchronously writes a batch of audit log rows to
	// audit_logs_raw_v1. Used by the AuditLogExport outbox worker — returns
	// only after ClickHouse confirms the insert so the caller can safely
	// mark the source MySQL rows as exported.
	InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error
}

type ClickHouse interface {
	Querier

	// Closes the underlying ClickHouse connection.
	Close() error

	// Ping verifies the connection to the ClickHouse database.
	Ping(ctx context.Context) error
}
