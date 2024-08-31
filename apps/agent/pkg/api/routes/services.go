package routes

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
)

type Services struct {
	Logger     logging.Logger
	Metrics    metrics.Metrics
	Vault      *vault.Service
	Ratelimit  ratelimit.Service
	Clickhouse clickhouse.Ingester
}
