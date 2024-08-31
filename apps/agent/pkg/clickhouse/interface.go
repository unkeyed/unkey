package clickhouse

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
)

type Ingester interface {
	InsertApiRequest(schema.ApiRequestV1)
}
