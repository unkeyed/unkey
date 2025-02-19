package clickhouse

import (
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

type Bufferer interface {
	BufferApiRequest(schema.ApiRequestV1)
	BufferKeyVerification(schema.KeyVerificationRequestV1)
}
