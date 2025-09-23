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
	// BufferRequest adds an API request event to the buffer.
	// These are typically HTTP requests to the API with request and response details.
	BufferRequest(schema.ApiRequestV1)

	// BufferApiRequest adds an API request event to the buffer.
	// These are typically HTTP requests to the API with request and response details.
	BufferApiRequest(schema.ApiRequestV2)

	// BufferKeyVerification adds a key verification event to the buffer.
	// These represent API key validation operations with their outcomes.
	BufferKeyVerification(schema.KeyVerificationRequestV1)

	// BufferRatelimit adds a ratelimit event to the buffer.
	// These represent API ratelimit operations with their outcome.
	BufferRatelimit(schema.RatelimitRequestV1)
}

type Querier interface {
	// Conn returns a connection to the ClickHouse database.
	Conn() ch.Conn

	GetBillableVerifications(ctx context.Context, workspaceID string, year, month int) (int64, error)
	GetBillableRatelimits(ctx context.Context, workspaceID string, year, month int) (int64, error)
}

type ClickHouse interface {
	Bufferer
	Querier
}
