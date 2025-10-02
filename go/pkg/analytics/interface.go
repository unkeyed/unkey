package analytics

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// Writer defines the interface for analytics event storage backends.
// Implementations can target different storage systems like ClickHouse,
// Kafka, or data lakes while providing a consistent API.
//
// Available implementations:
//   - analytics/storage/clickhouse: Direct ClickHouse storage
//   - analytics/storage/kafka: Kafka message publishing
//   - analytics/storage/iceberg: Customer-specific data lakes
type Writer interface {
	// KeyVerification writes a key verification event to the storage backend.
	// This represents an API key validation operation with its outcome.
	KeyVerification(ctx context.Context, data schema.KeyVerificationV2) error

	// Ratelimit writes a ratelimit event to the storage backend.
	// This represents a rate limiting operation with its outcome.
	Ratelimit(ctx context.Context, data schema.RatelimitV2) error

	// ApiRequest writes an API request event to the storage backend.
	// This represents an HTTP API request with request and response details.
	ApiRequest(ctx context.Context, data schema.ApiRequestV2) error

	// Close gracefully shuts down the writer, ensuring any pending
	// writes are completed before closing.
	Close(ctx context.Context) error
}
