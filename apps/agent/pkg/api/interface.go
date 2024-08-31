package api

import "github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"

type EventIngester interface {
	InsertApiRequest(schema.ApiRequestV1)
}
