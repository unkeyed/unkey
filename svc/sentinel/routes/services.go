package routes

import (
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

// Services contains all dependencies needed by route handlers
type Services struct {
	Logger             logging.Logger
	RouterService      router.Service
	Clock              clock.Clock
	WorkspaceID        string
	EnvironmentID      string
	SentinelID         string
	Region             string
	ClickHouse         clickhouse.ClickHouse
	MaxRequestBodySize int64

	// --- Wide configuration ---

	// WideSuccessSampleRate is the sampling rate for successful requests (0.0 - 1.0).
	WideSuccessSampleRate float64

	// WideSlowThresholdMs is the threshold in milliseconds for slow request logging.
	WideSlowThresholdMs int

	// Image is the service version/image identifier.
	Image string
}
