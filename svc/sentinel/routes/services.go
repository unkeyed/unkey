package routes

import (
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/middleware"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

// Services contains all dependencies needed by route handlers
type Services struct {
	RouterService      router.Service
	Clock              clock.Clock
	WorkspaceID        string
	EnvironmentID      string
	SentinelID         string
	Region             string
	ClickHouse         clickhouse.ClickHouse
	MaxRequestBodySize int64
	ZenMetrics         zen.Metrics
	ObsMetrics         middleware.Metrics
}
