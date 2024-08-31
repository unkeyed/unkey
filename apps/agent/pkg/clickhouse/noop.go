package clickhouse

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
)

type noop struct{}

var _ Ingester = &noop{}

func (n *noop) InsertApiRequest(schema.ApiRequestV1) {
	return
}

func NewNoopIngester() Ingester {
	return &noop{}
}
