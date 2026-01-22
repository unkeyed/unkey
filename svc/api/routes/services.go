package routes

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/zen/validation"
)

type Services struct {
	Logger                     logging.Logger
	Database                   db.Database
	Keys                       keys.KeyService
	ClickHouse                 clickhouse.ClickHouse
	Validator                  *validation.Validator
	Ratelimit                  ratelimit.Service
	Auditlogs                  auditlogs.AuditLogService
	Caches                     caches.Caches
	Vault                      *vault.Service
	ChproxyToken               string
	CtrlDeploymentClient       ctrlv1connect.DeploymentServiceClient
	PprofEnabled               bool
	PprofUsername              string
	PprofPassword              string
	UsageLimiter               usagelimiter.Service
	AnalyticsConnectionManager analytics.ConnectionManager
}
