package routes

import (
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/sentinel/services/router"
)

// Services contains all dependencies needed by route handlers
type Services struct {
	Logger        logging.Logger
	RouterService router.Service
	Clock         clock.Clock
	EnvironmentID string
	Region        string
}
