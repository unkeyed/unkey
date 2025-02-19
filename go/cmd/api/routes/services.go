package routes

import (
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}

type Services struct {
	Logger      logging.Logger
	Database    database.Database
	EventBuffer EventBuffer
	Keys        keys.KeyService
	Validator   *validation.Validator
	Ratelimit   ratelimit.Service
}
