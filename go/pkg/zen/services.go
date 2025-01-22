package zen

import (
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}

type Services struct {
	Logger      logging.Logger
	Database    database.Database
	EventBuffer EventBuffer
}
