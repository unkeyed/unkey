package routes

import (
	"github.com/unkeyed/unkey/go/apps/gateway/services/router"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Services contains all dependencies needed by route handlers
type Services struct {
	Logger        logging.Logger
	RouterService router.Service
	Clock         clock.Clock
	EnvironmentID string
	Region        string
}
