package router

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Service routes requests to deployment instances
type Service interface {
	// GetDeployment returns the deployment for the given deployment ID
	// Validates that the deployment belongs to this gateway's environment
	GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error)

	// SelectInstance returns a healthy instance for the deployment in this region
	SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error)
}

// Config holds configuration for the router service
type Config struct {
	Logger        logging.Logger
	DB            db.Database
	Clock         clock.Clock
	EnvironmentID string // Environment this gateway serves
	Region        string // Region this gateway runs in
}
