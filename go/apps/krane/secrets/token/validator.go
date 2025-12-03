package token

import (
	"context"
)

// ValidationResult contains information about the validated token
type ValidationResult struct {
	// DeploymentID is the deployment this token is valid for
	DeploymentID string
}

// Validator validates tokens used to fetch deployment secrets.
// Different implementations support different authentication mechanisms.
type Validator interface {
	// Validate checks if a token is valid for fetching secrets.
	// Returns ValidationResult with deployment info if valid, error otherwise.
	Validate(ctx context.Context, token string, deploymentID string) (*ValidationResult, error)
}

// Generator generates tokens for deployments.
// Used by the krane when creating pods/containers.
type Generator interface {
	// Generate creates a new token for a deployment.
	// Returns the token string that should be passed to the workload.
	Generate(ctx context.Context, deploymentID string, environmentID string) (string, error)
}

// ValidatorGenerator combines both interfaces for implementations that support both.
type ValidatorGenerator interface {
	Validator
	Generator
}
