package api

import "github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}
