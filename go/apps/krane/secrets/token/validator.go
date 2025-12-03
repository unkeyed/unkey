package token

import "context"

type ValidationResult struct {
	DeploymentID string
}

type Validator interface {
	Validate(ctx context.Context, token string, deploymentID string) (*ValidationResult, error)
}
