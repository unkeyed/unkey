package server

import "github.com/unkeyed/unkey/go/pkg/clickhouse/schema"

// EventBuffer is a sink to emit various events
// events will be batched and flushed later
type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}
