package router

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Deployment contains the routing target and configuration
type Deployment struct {
	// ID is the deployment identifier
	ID string

	// TargetAddress is the k8s service address to proxy to
	TargetAddress string

	// Config holds middleware and other deployment settings
	// TODO: define middleware types when ready
	Config DeploymentConfig
}

// DeploymentConfig holds middleware and other deployment settings
type DeploymentConfig struct {
	// Middlewares to apply to requests
	// TODO: define concrete middleware types
	Middlewares []any

	// Timeout for requests to this deployment
	TimeoutMs int

	// Other config as needed...
}

// Service resolves deployment IDs to their full deployment config
type Service interface {
	// GetDeployment returns the deployment config including target address and middlewares
	GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error)
}

// Config holds configuration for the router service
type Config struct {
	Logger logging.Logger
	DB     db.Database
	Clock  clock.Clock
}
