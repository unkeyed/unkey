package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	sentineldb "github.com/unkeyed/unkey/svc/sentinel/db"
)

type Service interface {
	GetDeployment(ctx context.Context, deploymentID string) (sentineldb.Deployment, error)
	SelectInstance(ctx context.Context, deploymentID string) (sentineldb.Instance, error)
}

type Config struct {
	Logger        logging.Logger
	DB            db.Database
	Clock         clock.Clock
	EnvironmentID string
	Region        string
}
