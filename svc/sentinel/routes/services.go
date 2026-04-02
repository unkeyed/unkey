package routes

import (
	"time"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
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
	Platform           string
	SentinelRequests   *batch.BatchProcessor[schema.SentinelRequest]
	MaxRequestBodySize int64
	RequestTimeout     time.Duration
	Engine             engine.Evaluator
	// Pprof enables pprof profiling endpoints when non-nil.
	Pprof *config.PprofConfig
}
