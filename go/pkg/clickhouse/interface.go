package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// Bufferer defines the interface for systems that can buffer events for
// batch processing. It provides methods to add different types of events
// to their respective buffers.
//
// This interface allows for different implementations, such as a real
// ClickHouse client or a no-op implementation for testing or development.
type Bufferer interface {
	// BufferApiRequest adds an API request event to the buffer.
	// These are typically HTTP requests to the API with request and response details.
	BufferApiRequest(schema.ApiRequest)

	// BufferKeyVerification adds a key verification event to the buffer.
	// These represent API key validation operations with their outcomes.
	BufferKeyVerification(schema.KeyVerification)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferRatelimit(schema.Ratelimit)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferBuildStep(schema.BuildStepV1)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferBuildStepLog(schema.BuildStepLogV1)
}

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
}

type ClickHouse interface {
	Bufferer
	Querier

	// Closes the underlying ClickHouse connection.
	Close() error

	// Ping verifies the connection to the ClickHouse database.
	Ping(ctx context.Context) error
}
