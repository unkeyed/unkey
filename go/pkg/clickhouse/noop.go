package clickhouse

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
)

type noop struct{}

var _ Bufferer = &noop{}

func (n *noop) BufferApiRequest(schema.ApiRequestV1) {

}
func (n *noop) BufferKeyVerification(schema.KeyVerificationRequestV1) {

}

func NewNoop() *noop {
	return &noop{}
}
