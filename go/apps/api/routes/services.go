package routes

import (
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}

type Services struct {
	Logger      logging.Logger
	Database    db.Database
	Keys        keys.KeyService
	ClickHouse  clickhouse.ClickHouse
	Permissions permissions.PermissionService
	Validator   *validation.Validator
	Ratelimit   ratelimit.Service
	Caches      caches.Caches
}
