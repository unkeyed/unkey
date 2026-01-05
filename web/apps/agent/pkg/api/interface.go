package api

import "github.com/unkeyed/unkey/svc/agent/pkg/clickhouse/schema"

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}
