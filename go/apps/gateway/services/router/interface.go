package router

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service interface {
	GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error)
	SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error)
}

type Config struct {
	Logger        logging.Logger
	DB            db.Database
	Clock         clock.Clock
	EnvironmentID string
	Region        string
}
