package clickhouse

import (
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
	BufferApiRequest(schema.ApiRequestV1)

	// BufferKeyVerification adds a key verification event to the buffer.
	// These represent API key validation operations with their outcomes.
	BufferKeyVerification(schema.KeyVerificationRequestV1)
}
