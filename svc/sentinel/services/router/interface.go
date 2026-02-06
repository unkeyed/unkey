package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service interface {
	GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error)
	SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error)
}

type Config struct {
	DB            db.Database
	Clock         clock.Clock
	EnvironmentID string
	Region        string
}
